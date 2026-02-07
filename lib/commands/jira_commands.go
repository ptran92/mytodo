package commands

import (
	"fmt"
	"mytodo/lib/jira"
	"mytodo/lib/quip"
	"mytodo/lib/utils"
	"os"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/spf13/cobra"
)

func NewJiraSummaryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "jira-summary",
		Short: "Show epic / task progress in JIRA",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !utils.AgentEnabled() {
				return fmt.Errorf("JIRA/AI not configured (missing env vars)")
			}
			// confirmation
			yes, err := llmAgent.AskConfirmation("Summarise JIRA project? (yes/no)")
			if err != nil || !yes {
				return err
			}

			client := jira.NewClient(utils.GetJiraURL(), utils.GetJiraEmail(), utils.GetJiraToken())
			project := utils.GetProjectKey()
			// 1. list epics
			epics, err := client.Search(fmt.Sprintf("project=%s AND type=Epic", project))
			if err != nil {
				return err
			}
			fmt.Printf("Project %s has %d epics\n", project, len(epics))

			// 2. for each epic, list stories & compute stats
			for _, epic := range epics {
				epicKey := epic["key"].(string)
				epicName := epic["fields"].(map[string]interface{})["summary"].(string)
				fmt.Printf("\nEpic %s: %s\n", epicKey, epicName)

				stories, err := client.Search(fmt.Sprintf("project=%s AND \"Epic Link\"=%s", project, epicKey))
				if err != nil {
					return err
				}

				var completed, pending, estPoints float64
				for _, s := range stories {
					status := s["fields"].(map[string]interface{})["status"].(map[string]interface{})["name"].(string)
					if status == "Done" {
						completed++
					} else {
						pending++
					}
					// assume story points field = customfield_10016
					if v, ok := s["fields"].(map[string]interface{})["customfield_10016"]; ok {
						if num, ok := v.(float64); ok {
							estPoints += num
							// For actual points, you might have another custom field
						}
					}
				}
				fmt.Printf("  %d stories: %d done, %d pending\n", len(stories), int(completed), int(pending))
				fmt.Printf("  Est. points: %.1f\n", estPoints)
				// … add time estimation logic if you have fields …
			}
			return nil
		},
	}
}

func NewJiraCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "jira-create",
		Short: "Create a new JIRA task with an optional label",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !utils.AgentEnabled() {
				return fmt.Errorf("JIRA/AI not configured")
			}
			yes, err := llmAgent.AskConfirmation("Create JIRA issue? (yes/no)")
			if err != nil || !yes {
				return err
			}

			var summary, desc, label string
			fmt.Print("Summary: ")
			fmt.Scanln(&summary)
			fmt.Print("Description: ")
			fmt.Scanln(&desc)
			fmt.Print("Label (optional): ")
			fmt.Scanln(&label)

			client := jira.NewClient(utils.GetJiraURL(), utils.GetJiraEmail(), utils.GetJiraToken())
			key, err := client.Create(summary, desc, "Task", []string{label})
			if err != nil {
				return err
			}
			fmt.Printf("Created %s\n", key)
			return nil
		},
	}
}

func NewJiraEpicTrackerCmd() *cobra.Command {
	var outputFormat string
	var quipDocURL string
	var outputFile string

	cmd := &cobra.Command{
		Use:   "jira-epic-tracker [epic-key]",
		Short: "Generate a project tracker table from a JIRA epic ticket",
		Long: `Query all stories, tasks, and bugs linked to a JIRA epic and display them in a tracker table format.
The table includes ticket ID, description, owner, status, estimates, and other metadata.

Example: mytodo jira-epic-tracker ASD-146
Example with Quip: mytodo jira-epic-tracker ASD-146 --quip "https://domain.quip.com/ABCD123/My-test-project"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			epicKey := args[0]

			// Check JIRA configuration
			jiraURL := utils.GetJiraURL()
			jiraEmail := utils.GetJiraEmail()
			jiraToken := utils.GetJiraToken()
			if jiraURL == "" || jiraEmail == "" || jiraToken == "" {
				return fmt.Errorf("JIRA not configured. Please set JIRA_URL, JIRA_EMAIL, and JIRA_TOKEN environment variables")
			}

			fmt.Printf("Fetching epic %s and linked issues...\n", epicKey)

			// Create JIRA client
			client := jira.NewClient(jiraURL, jiraEmail, jiraToken)

			// Get the epic details
			epicIssue, err := client.GetIssue(epicKey)
			if err != nil {
				return fmt.Errorf("failed to fetch epic: %w", err)
			}

			epicFields, ok := epicIssue["fields"].(map[string]interface{})
			if !ok {
				return fmt.Errorf("invalid epic response format")
			}

			epicName := ""
			if summary, ok := epicFields["summary"].(string); ok {
				epicName = summary
			}

			fmt.Printf("Epic: %s - %s\n", epicKey, epicName)

			// Get all issues linked to the epic (children, subtasks, and linked items)
			issues, err := client.GetIssuesByEpic(epicKey)
			if err != nil {
				return fmt.Errorf("failed to fetch linked issues: %w", err)
			}

			fmt.Printf("Found %d related issues (child work items, subtasks, and linked issues)\n\n", len(issues))

			if len(issues) == 0 {
				fmt.Println("No issues related to this epic")
				return nil
			}

			// Count issue types for informational purposes
			typeCount := make(map[string]int)
			for _, issue := range issues {
				if fields, ok := issue["fields"].(map[string]interface{}); ok {
					if issueType, ok := fields["issuetype"].(map[string]interface{}); ok {
						if typeName, ok := issueType["name"].(string); ok {
							typeCount[typeName]++
						}
					}
				}
			}

			fmt.Println("Issue breakdown by type:")
			for typeName, count := range typeCount {
				fmt.Printf("  - %s: %d\n", typeName, count)
			}
			fmt.Println()

			// Convert issues to tracker rows
			var rows []*jira.TicketRow
			for _, issue := range issues {
				if row := jira.ExtractTicketRow(issue); row != nil {
					rows = append(rows, row)
				}
			}

			// Format output based on requested format
			var output string
			switch outputFormat {
			case "csv":
				output = jira.FormatAsCSV(rows)
			case "markdown", "md":
				output = jira.FormatAsMarkdownTable(rows, epicKey, epicName)
			default:
				output = jira.FormatAsMarkdownTable(rows, epicKey, epicName)
			}

			// Display output
			fmt.Println(output)

			// Save to file if requested
			if outputFile != "" {
				if err := os.WriteFile(outputFile, []byte(output), 0644); err != nil {
					return fmt.Errorf("failed to write output file: %w", err)
				}
				fmt.Printf("\n✅ Saved to %s\n", outputFile)
			}

			// Append to Quip if URL is provided
			if quipDocURL != "" {
				quipToken := utils.GetQuipToken()
				if quipToken == "" {
					return fmt.Errorf("Quip not configured. Please set QUIP_TOKEN environment variable")
				}

				quipClient := quip.NewClient(quipToken)

				// Extract thread ID from Quip URL
				// Format: https://domain.quip.com/9asfn23ujd/name-of-document
				// Thread ID is the part after the domain before the slash
				threadID, err := extractQuipThreadID(quipDocURL)
				if err != nil {
					return fmt.Errorf("invalid Quip URL: %w", err)
				}

				fmt.Printf("\nAppending to Quip document: %s\n", quipDocURL)

				// Prepare content with separator
				separator := fmt.Sprintf("\n\n---\n**Updated: %s**\n\n", epicKey)
				contentToAppend := separator + output

				// Try both markdown and HTML formats
				err = quipClient.AppendTableToDocument(threadID, markdownToHTML(contentToAppend))
				if err != nil {
					// If HTML fails, try markdown format directly
					fmt.Printf("HTML append failed, trying markdown format: %v\n", err)
					err = quipClient.AppendToDocument(threadID, contentToAppend)
					if err != nil {
						return fmt.Errorf("failed to append to Quip document: %w", err)
					}
				}

				fmt.Printf("✅ Content appended to Quip document: %s\n", quipDocURL)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "format", "f", "markdown", "Output format: markdown, csv")
	cmd.Flags().StringVarP(&quipDocURL, "quip", "q", "", "Quip document URL to append to (e.g., https://domain.quip.com/ABC123/Document-Name)")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Save output to file")

	return cmd
}

// extractQuipThreadID extracts the thread ID from a Quip URL
// Format: https://domain.quip.com/ePhzA3UgR8Wd/My-test-project
// Thread ID: ePhzA3UgR8Wd
func extractQuipThreadID(quipURL string) (string, error) {
	// Remove trailing slash if present
	quipURL = strings.TrimSuffix(quipURL, "/")

	// Parse the URL
	parts := strings.Split(quipURL, "/")

	// Find the thread ID (should be after the domain)
	// URL format: https://domain.quip.com/THREAD_ID/optional-title
	for i, part := range parts {
		if strings.Contains(part, "quip.com") && i+1 < len(parts) {
			threadID := parts[i+1]
			// Thread ID should be alphanumeric
			if len(threadID) > 0 {
				return threadID, nil
			}
		}
	}

	return "", fmt.Errorf("could not extract thread ID from URL: %s", quipURL)
}

// markdownToHTML converts Markdown content to HTML
func markdownToHTML(md string) string {
	// Create markdown parser with extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.Tables
	p := parser.NewWithExtensions(extensions)

	// Parse the markdown
	doc := p.Parse([]byte(md))

	// Create HTML renderer with extensions
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	// Render to HTML
	return string(markdown.Render(doc, renderer))
}
