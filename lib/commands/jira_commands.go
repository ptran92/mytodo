package commands

import (
	"fmt"
	"mytodo/lib/jira"
	"mytodo/lib/utils"

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

			client := jira.NewClient(utils.GetJiraURL(), utils.GetJiraToken())
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

			client := jira.NewClient(utils.GetJiraURL(), utils.GetJiraToken())
			key, err := client.Create(summary, desc, "Task", []string{label})
			if err != nil {
				return err
			}
			fmt.Printf("Created %s\n", key)
			return nil
		},
	}
}
