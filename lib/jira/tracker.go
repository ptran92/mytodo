package jira

import (
	"fmt"
	"strings"
	"time"
)

// TicketRow represents a single row in the project tracker table
type TicketRow struct {
	Ticket              string
	Description         string
	Owner               string
	ExpectedQADate      string
	PercentCompletion   float64
	PercentWeight       float64
	DaysToQAEstimated   float64
	DaysToQAActual      float64
	ReqNumber           string
	Status              string
	ExtDependencies     string
}

// ExtractTicketRow extracts tracker information from a JIRA issue
func ExtractTicketRow(issue map[string]interface{}) *TicketRow {
	fields, ok := issue["fields"].(map[string]interface{})
	if !ok {
		return nil
	}

	row := &TicketRow{}

	// Extract ticket key and type
	if key, ok := issue["key"].(string); ok {
		issueType := "unknown"
		if it, ok := fields["issuetype"].(map[string]interface{}); ok {
			if name, ok := it["name"].(string); ok {
				issueType = strings.ToLower(name)
			}
		}
		row.Ticket = fmt.Sprintf("%s (%s)", key, issueType)
	}

	// Extract description (summary)
	if summary, ok := fields["summary"].(string); ok {
		row.Description = summary
	}

	// Extract owner (assignee)
	if assignee, ok := fields["assignee"].(map[string]interface{}); ok {
		if displayName, ok := assignee["displayName"].(string); ok {
			row.Owner = displayName
		}
	}

	// Extract status
	if status, ok := fields["status"].(map[string]interface{}); ok {
		if statusName, ok := status["name"].(string); ok {
			row.Status = statusName
		}
	}

	// Extract custom fields (these may vary by JIRA configuration)
	// Expected QA Date - typically a custom field like customfield_XXXXX
	if qaDate, ok := fields["customfield_10020"].(string); ok {
		row.ExpectedQADate = qaDate
	}

	// Story points - the custom field ID varies by JIRA instance
	// Common IDs: customfield_10013, customfield_10016, customfield_10024
	// Try customfield_10013 first (your instance's story point field)
	if storyPoints, ok := fields["customfield_10013"].(float64); ok {
		row.DaysToQAEstimated = storyPoints
	} else if storyPoints, ok := fields["customfield_10016"].(float64); ok {
		// Fallback to another common story point field
		row.DaysToQAEstimated = storyPoints
	}

	// Percent completion based on status
	row.PercentCompletion = calculateCompletion(row.Status)

	// Extract dependencies from description or custom field
	if desc, ok := fields["description"].(string); ok {
		if strings.Contains(strings.ToLower(desc), "depend") {
			row.ExtDependencies = extractDependencies(desc)
		}
	}

	return row
}

// calculateCompletion estimates completion percentage based on status
func calculateCompletion(status string) float64 {
	status = strings.ToLower(status)
	switch {
	case strings.Contains(status, "done") || strings.Contains(status, "closed") || strings.Contains(status, "complete"):
		return 100.0
	case strings.Contains(status, "review") || strings.Contains(status, "qa"):
		return 90.0
	case strings.Contains(status, "progress"):
		return 50.0
	case strings.Contains(status, "started"):
		return 10.0
	default:
		return 0.0
	}
}

// extractDependencies extracts dependency information from text
func extractDependencies(text string) string {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "depend") {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

// FormatAsMarkdownTable formats ticket rows as a markdown table
func FormatAsMarkdownTable(rows []*TicketRow, epicKey, epicName string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Project Tracker: %s - %s\n\n", epicKey, epicName))
	sb.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// Table header
	sb.WriteString("| Ticket | Description | Owner | Expected QA Date | % Completion | % Weight | Days to QA (Est) | Days to QA (Actual) | Req # | Status | Ext Dependencies |\n")
	sb.WriteString("|--------|-------------|-------|------------------|--------------|----------|------------------|---------------------|-------|--------|------------------|\n")

	// Calculate total estimated days for weight calculation
	totalDays := 0.0
	for _, row := range rows {
		totalDays += row.DaysToQAEstimated
	}

	// Table rows
	for _, row := range rows {
		if totalDays > 0 {
			row.PercentWeight = (row.DaysToQAEstimated / totalDays) * 100.0
		}

		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %.2f%% | %.2f%% | %.1f | %.1f | %s | %s | %s |\n",
			row.Ticket,
			row.Description,
			row.Owner,
			row.ExpectedQADate,
			row.PercentCompletion,
			row.PercentWeight,
			row.DaysToQAEstimated,
			row.DaysToQAActual,
			row.ReqNumber,
			row.Status,
			row.ExtDependencies,
		))
	}

	// Summary statistics
	sb.WriteString(fmt.Sprintf("\n## Summary\n"))
	sb.WriteString(fmt.Sprintf("- Total tickets: %d\n", len(rows)))

	completed := 0
	inProgress := 0
	notStarted := 0
	for _, row := range rows {
		if row.PercentCompletion >= 100 {
			completed++
		} else if row.PercentCompletion > 0 {
			inProgress++
		} else {
			notStarted++
		}
	}

	sb.WriteString(fmt.Sprintf("- Completed: %d\n", completed))
	sb.WriteString(fmt.Sprintf("- In Progress: %d\n", inProgress))
	sb.WriteString(fmt.Sprintf("- Not Started: %d\n", notStarted))
	sb.WriteString(fmt.Sprintf("- Total Estimated Days: %.1f\n", totalDays))

	if len(rows) > 0 {
		overallCompletion := 0.0
		for _, row := range rows {
			overallCompletion += row.PercentCompletion * row.PercentWeight / 100.0
		}
		sb.WriteString(fmt.Sprintf("- Overall Completion: %.2f%%\n", overallCompletion))
	}

	return sb.String()
}

// FormatAsCSV formats ticket rows as CSV
func FormatAsCSV(rows []*TicketRow) string {
	var sb strings.Builder

	// CSV header
	sb.WriteString("Ticket,Description,Owner,Expected QA Date,% Completion,% Weight,Days to QA (Est),Days to QA (Actual),Req #,Status,Ext Dependencies\n")

	// Calculate total estimated days for weight calculation
	totalDays := 0.0
	for _, row := range rows {
		totalDays += row.DaysToQAEstimated
	}

	// CSV rows
	for _, row := range rows {
		if totalDays > 0 {
			row.PercentWeight = (row.DaysToQAEstimated / totalDays) * 100.0
		}

		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s,%.2f,%.2f,%.1f,%.1f,%s,%s,%s\n",
			escapeCSV(row.Ticket),
			escapeCSV(row.Description),
			escapeCSV(row.Owner),
			escapeCSV(row.ExpectedQADate),
			row.PercentCompletion,
			row.PercentWeight,
			row.DaysToQAEstimated,
			row.DaysToQAActual,
			escapeCSV(row.ReqNumber),
			escapeCSV(row.Status),
			escapeCSV(row.ExtDependencies),
		))
	}

	return sb.String()
}

// escapeCSV escapes CSV values that contain commas or quotes
func escapeCSV(value string) string {
	if strings.Contains(value, ",") || strings.Contains(value, "\"") || strings.Contains(value, "\n") {
		return "\"" + strings.ReplaceAll(value, "\"", "\"\"") + "\""
	}
	return value
}
