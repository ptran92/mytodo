package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	v3 "github.com/ctreminiom/go-atlassian/jira/v3"
	"github.com/ctreminiom/go-atlassian/pkg/infra/models"
)

// Type aliases for easier use
type IssueScheme = models.IssueScheme
type IssueFieldsScheme = models.IssueFieldsScheme
type ProjectScheme = models.ProjectScheme
type IssueTypeScheme = models.IssueTypeScheme

type Client struct {
	client   *v3.Client
	ctx      context.Context
	baseURL  string
	email    string
	apiToken string
}

func NewClient(baseURL, email, token string) *Client {
	// Create atlassian client with basic auth
	client, err := v3.New(nil, baseURL)
	if err != nil {
		panic(fmt.Sprintf("failed to create JIRA client: %v", err))
	}

	// Set basic authentication
	client.Auth.SetBasicAuth(email, token)

	return &Client{
		client:   client,
		ctx:      context.Background(),
		baseURL:  baseURL,
		email:    email,
		apiToken: token,
	}
}

// searchJQL uses the new /rest/api/3/search/jql endpoint directly
func (c *Client) searchJQL(jql string, startAt, maxResults int) (*models.IssueSearchScheme, error) {
	// Build the URL
	u, err := url.Parse(c.baseURL + "/rest/api/3/search/jql")
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Add query parameters
	q := u.Query()
	q.Set("jql", jql)
	q.Set("startAt", fmt.Sprintf("%d", startAt))
	q.Set("maxResults", fmt.Sprintf("%d", maxResults))
	q.Set("fields", "*all")
	u.RawQuery = q.Encode()

	// Create HTTP request
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set basic auth
	req.SetBasicAuth(c.email, c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result models.IssueSearchScheme
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// Search JQL
func (c *Client) Search(jql string) ([]map[string]interface{}, error) {
	result, err := c.searchJQL(jql, 0, 100)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Convert to map[string]interface{} for compatibility
	var issues []map[string]interface{}
	for _, issue := range result.Issues {
		issueMap := convertIssueToMap(issue)
		issues = append(issues, issueMap)
	}

	return issues, nil
}

// Create an issue
func (c *Client) Create(summary, desc, issueType string, tags []string) (string, error) {
	payload := &IssueScheme{
		Fields: &IssueFieldsScheme{
			Summary: summary,
			Project: &ProjectScheme{
				Key: "PROJ",
			},
			IssueType: &IssueTypeScheme{
				Name: issueType,
			},
		},
	}

	if len(tags) > 0 {
		payload.Fields.Labels = tags
	}

	newIssue, _, err := c.client.Issue.Create(c.ctx, payload, nil)
	if err != nil {
		return "", fmt.Errorf("create failed: %w", err)
	}

	return newIssue.Key, nil
}

// GetIssuesByEpic fetches all issues linked to an epic, including:
// - Child issues (parent = epic)
// - Epic Link issues (Epic Link = epic)
// - Subtasks
func (c *Client) GetIssuesByEpic(epicKey string) ([]map[string]interface{}, error) {
	issueMap := make(map[string]map[string]interface{})

	// Strategy 1: Search for child issues using parent field (modern JIRA)
	jql := fmt.Sprintf("parent = %s", epicKey)
	fmt.Printf("Searching with JQL: %s\n", jql)

	// Handle pagination - fetch all pages
	startAt := 0
	maxResults := 100

	for {
		// Use the new JQL endpoint
		result, err := c.searchJQL(jql, startAt, maxResults)
		if err != nil {
			fmt.Printf("Warning: JQL search with 'parent' failed: %v\n", err)
			break
		}

		for _, issue := range result.Issues {
			issueMap[issue.Key] = convertIssueToMap(issue)
		}

		// Check if we've fetched all results
		if startAt+len(result.Issues) >= result.Total {
			break
		}
		startAt += maxResults
	}

	fmt.Printf("Found %d issues with 'parent' field\n", len(issueMap))

	// Strategy 2: Try Epic Link field (legacy JIRA - field name varies by instance)
	// Common Epic Link custom fields: customfield_10014, customfield_10008
	epicLinkFields := []string{
		"\"Epic Link\"",
		"cf[10014]",
		"cf[10008]",
	}

	for _, epicLinkField := range epicLinkFields {
		jql2 := fmt.Sprintf("%s = %s", epicLinkField, epicKey)
		fmt.Printf("Trying JQL: %s\n", jql2)

		startAt = 0
		for {
			// Use the new JQL endpoint
			result, err := c.searchJQL(jql2, startAt, maxResults)
			if err != nil {
				// This field might not exist, try next one
				fmt.Printf("Warning: JQL search with '%s' failed: %v\n", epicLinkField, err)
				break
			}

			foundInThisQuery := 0
			for _, issue := range result.Issues {
				if _, exists := issueMap[issue.Key]; !exists {
					issueMap[issue.Key] = convertIssueToMap(issue)
					foundInThisQuery++
				}
			}

			fmt.Printf("Found %d new issues with '%s' field\n", foundInThisQuery, epicLinkField)

			// Check if we've fetched all results
			if startAt+len(result.Issues) >= result.Total {
				break
			}
			startAt += maxResults
		}
	}

	// Strategy 3: Get the epic itself and extract subtasks
	epic, err := c.GetIssue(epicKey)
	if err == nil {
		if fields, ok := epic["fields"].(map[string]interface{}); ok {
			if subtasks, ok := fields["subtasks"].([]interface{}); ok {
				for _, subtask := range subtasks {
					if st, ok := subtask.(map[string]interface{}); ok {
						if key, ok := st["key"].(string); ok {
							if _, exists := issueMap[key]; !exists {
								// Fetch full details for subtask
								fullSubtask, err := c.GetIssue(key)
								if err == nil {
									issueMap[key] = fullSubtask
								}
							}
						}
					}
				}
			}
		}
	}

	// Convert map to slice
	var result []map[string]interface{}
	for _, issue := range issueMap {
		result = append(result, issue)
	}

	fmt.Printf("Total unique issues found: %d\n", len(result))
	return result, nil
}

// GetIssue fetches a single issue with all fields
func (c *Client) GetIssue(issueKey string) (map[string]interface{}, error) {
	fields := []string{"*all"}
	expand := []string{"renderedFields", "names", "schema", "transitions"}

	issue, _, err := c.client.Issue.Get(c.ctx, issueKey, fields, expand)
	if err != nil {
		return nil, fmt.Errorf("get issue failed: %w", err)
	}

	return convertIssueToMap(issue), nil
}

// Helper function to convert go-atlassian IssueScheme to map[string]interface{}
func convertIssueToMap(issue *IssueScheme) map[string]interface{} {
	result := make(map[string]interface{})

	result["key"] = issue.Key
	result["id"] = issue.ID

	// Convert fields
	if issue.Fields != nil {
		fields := make(map[string]interface{})

		fields["summary"] = issue.Fields.Summary
		fields["description"] = issue.Fields.Description

		// Assignee
		if issue.Fields.Assignee != nil {
			fields["assignee"] = map[string]interface{}{
				"displayName": issue.Fields.Assignee.DisplayName,
				"emailAddress": issue.Fields.Assignee.EmailAddress,
			}
		}

		// Status
		if issue.Fields.Status != nil {
			fields["status"] = map[string]interface{}{
				"name": issue.Fields.Status.Name,
				"id":   issue.Fields.Status.ID,
			}
		}

		// Issue Type
		if issue.Fields.IssueType != nil {
			fields["issuetype"] = map[string]interface{}{
				"name": issue.Fields.IssueType.Name,
				"id":   issue.Fields.IssueType.ID,
			}
		}

		// Priority
		if issue.Fields.Priority != nil {
			fields["priority"] = map[string]interface{}{
				"name": issue.Fields.Priority.Name,
				"id":   issue.Fields.Priority.ID,
			}
		}

		// Created/Updated dates (these are strings in the go-atlassian models)
		if issue.Fields.Created != "" {
			fields["created"] = issue.Fields.Created
		}
		if issue.Fields.Updated != "" {
			fields["updated"] = issue.Fields.Updated
		}

		// Subtasks
		if len(issue.Fields.Subtasks) > 0 {
			var subtasks []interface{}
			for _, st := range issue.Fields.Subtasks {
				subtasks = append(subtasks, map[string]interface{}{
					"key": st.Key,
					"id":  st.ID,
				})
			}
			fields["subtasks"] = subtasks
		}

		// Issue links
		if len(issue.Fields.IssueLinks) > 0 {
			var links []interface{}
			for _, link := range issue.Fields.IssueLinks {
				linkMap := make(map[string]interface{})
				if link.InwardIssue != nil {
					linkMap["inwardIssue"] = map[string]interface{}{
						"key": link.InwardIssue.Key,
					}
				}
				if link.OutwardIssue != nil {
					linkMap["outwardIssue"] = map[string]interface{}{
						"key": link.OutwardIssue.Key,
					}
				}
				links = append(links, linkMap)
			}
			fields["issuelinks"] = links
		}

		// Labels
		if len(issue.Fields.Labels) > 0 {
			fields["labels"] = issue.Fields.Labels
		}

		result["fields"] = fields
	}

	return result
}
