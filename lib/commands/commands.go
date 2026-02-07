package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mytodo/lib/agent"
	"mytodo/lib/tasklist"
	"mytodo/lib/utils"
	"strings"

	"os"
	"strconv"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	MasterTasks *tasklist.TaskList
	llmAgent    agent.LlmAgent
)

func SetMasterTasks(t *tasklist.TaskList) {
	MasterTasks = t
}

// SetAgent allows cmd/main to inject the LLM agent into the commands package.
func SetAgent(a agent.LlmAgent) {
	llmAgent = a
}

func nicePrint(writer io.Writer, tasks []tasklist.Task) error {
	// Define styled printers
	success := color.New(color.FgGreen, color.Bold).Sprintf
	info := color.New(color.FgCyan).Sprintf

	commentPrinter := func(comments []string) {
		for _, comment := range comments {
			fmt.Fprintf(writer, "\t\t- %s\n", comment)
		}
	}

	// Write each task with appropriate style and icon
	for index, task := range tasks {
		var formatted string
		switch task.Done {
		case true:
			formatted = success("✔\t%d. %s: %s", index, task.Content, "Completed")
		case false:
			formatted = info("⏳\t%d. %s: %s", index, task.Content, "Pending")
		}
		if _, err := fmt.Fprintln(writer, formatted); err != nil {
			return fmt.Errorf("failed to write task %s: %w", task.Content, err)
		}
		commentPrinter(task.Comments)
	}
	return nil
}

func GetTaskList() *tasklist.TaskList {
	return MasterTasks
}

func PrepareCommands() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "mytodo",
		Short: "Manage your TODOs",
	}

	var verbose bool
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	addCmd := createAddCmd(verbose)

	listCmd := createListCmd(verbose)

	removeCommand := &cobra.Command{
		Use:   "remove [task number]",
		Short: "Remove a task by its number",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if GetTaskList().NumberOfTasks() == 0 {
				fmt.Println("No tasks to remove.")
				return
			}
			defer printToStdout()

			id, err := indexFromArgument(args)
			if err != nil {
				return
			}

			if verbose {
				fmt.Println("Removing task with ID:", id)
			}

			GetTaskList().RemoveTask(id)

		},
	}

	doneCommand := &cobra.Command{
		Use:   "done [task number]",
		Short: "Mark a task as done by its number",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(GetTaskList().Tasks) == 0 {
				fmt.Println("No tasks to mark as done.")
				return
			}
			defer printToStdout()

			id, err := indexFromArgument(args)
			if err != nil {
				return
			}

			if verbose {
				fmt.Println("Marking task with ID as done:", id)
			}

			t := GetTaskList().GetTask(id)
			t.Done = true
			GetTaskList().ReplaceTask(id, t)

		},
	}

	undoneCommand := &cobra.Command{
		Use:   "undone [task number]",
		Short: "Mark a task as not done by its number",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(GetTaskList().Tasks) == 0 {
				fmt.Println("No tasks to mark as not done.")
				return
			}
			defer printToStdout()

			id, err := indexFromArgument(args)
			if err != nil {
				return
			}

			if verbose {
				fmt.Println("Marking task with ID as not done:", id)
			}

			t := GetTaskList().GetTask(id)
			t.Done = false
			GetTaskList().ReplaceTask(id, t)
		},
	}

	editCommand := &cobra.Command{
		Use:   "edit [task number] [new content]",
		Short: "Edit a task's content by its number",
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			if len(GetTaskList().Tasks) == 0 {
				fmt.Println("No tasks to edit.")
				return
			}
			defer printToStdout()

			id, err := indexFromArgument(args)
			if err != nil {
				return
			}

			newContent := args[1]
			if verbose {
				fmt.Println("Editing task with ID:", id, "to new content:", newContent)
			}

			t := GetTaskList().GetTask(id)
			t.Content = newContent
			GetTaskList().ReplaceTask(id, t)
		},
	}

	addComment := &cobra.Command{
		Use:   "cm [task number] [comment]",
		Short: "Add a comment to a task by its number",
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			if GetTaskList().NumberOfTasks() == 0 {
				fmt.Println("No tasks to comment on.")
				return
			}
			defer printToStdout()

			id, err := indexFromArgument(args)
			if err != nil {
				return
			}

			comment := args[1]
			GetTaskList().AddComment(id, comment)
		},
	}

	jiraSummaryCmd := NewJiraSummaryCmd()

	jiraCreateCmd := NewJiraCreateCmd()

	jiraEpicTrackerCmd := NewJiraEpicTrackerCmd()

	rootCmd.AddCommand(
		addCmd,
		listCmd,
		removeCommand,
		doneCommand,
		undoneCommand,
		editCommand,
		addComment,
		jiraSummaryCmd,
		jiraCreateCmd,
		jiraEpicTrackerCmd,
	)
	return rootCmd
}

func indexFromArgument(args []string) (int, error) {
	id, err := strconv.Atoi(args[0])
	if err != nil || id < 0 || id >= len(GetTaskList().Tasks) {
		fmt.Println("Invalid task number.")
		return -1, fmt.Errorf("invalid task number")
	}
	return id, nil
}

func printToStdout() {
	tasks := GetTaskList().GetAllTasks()
	nicePrint(os.Stdout, tasks)
}

func createListCmd(verbose bool) *cobra.Command {
	var summary bool
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			if GetTaskList().NumberOfTasks() == 0 {
				fmt.Println("No tasks found.")
				return nil
			}

			// Print them to terminal directly
			printToStdout()

			// Summarize if set
			if summary {
				// 1️⃣  Gather all tasks
				tasks := GetTaskList().GetAllTasks()

				// 2️⃣  Serialize to JSON (used by the LLM prompt)
				b, err := json.Marshal(tasks)
				if err != nil {
					return fmt.Errorf("serializing tasks: %w", err)
				}

				// 3️⃣  Build the prompt
				summaryPrompt := fmt.Sprintf(`Here is the list of tasks in JSON:
		%s
		Summarize the above list in one concise sentence.`, string(b))

				sResp, err := llmAgent.Prompt(summaryPrompt)
				if err != nil {
					return fmt.Errorf("LLM summary prompt error: %w", err)
				}
				summaryText, err := sResp.GetResponse()
				if err != nil {
					return fmt.Errorf("reading LLM summary: %w", err)
				}
				fmt.Println("\n🔍 Summary:", strings.TrimSpace(summaryText))
			}

			return nil
		},
	}
	listCmd.Flags().BoolVarP(&summary, "summary", "s", false, "Show a short summary of the tasks")
	return listCmd
}

func createAddCmd(verbose bool) *cobra.Command {
	return &cobra.Command{
		Use:   "add",
		Short: "Create tasks from free‑form text via the LLM",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !utils.AgentEnabled() {
				// default to no AI mode
				todo := args[0]
				if verbose {
					fmt.Println("Adding task:", todo)
				}

				task := tasklist.Task{
					Content: todo,
					Done:    false,
				}

				GetTaskList().AddTask(&task)
				return nil
			}

			// Otherwise use AI agent to generate tasks
			// ① Gather the user’s free‑form thoughts
			var rawInput string
			if len(args) > 0 {
				// Arguments are taken as a single string
				rawInput = strings.Join(args, " ")
			} else {
				// If nothing was passed, read from STDIN
				data, err := ioutil.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("reading stdin: %w", err)
				}
				rawInput = string(data)
			}
			if rawInput == "" {
				return fmt.Errorf("no input supplied – give a sentence or pipe in text")
			}

			if verbose {
				fmt.Printf("Input: %v\n", rawInput)
			}

			// ② Ask the LLM to transform it into structured tasks
			response, err := llmAgent.Prompt(fmt.Sprintf(
				`Please turn the following note into a JSON array of tasks.  
Each task must have "content" (string) and "done" (boolean) fields.  
No extra keys. No explaination.

Note: "%s"`, rawInput))
			if err != nil {
				return fmt.Errorf("LLM prompt error: %w", err)
			}

			// ③ Grab the raw JSON string from the response
			resp, err := response.GetResponse()
			if err != nil {
				return fmt.Errorf("parsing LLM output: %w", err)
			}

			// ④ Pretty‑print the raw JSON for the user (optional)
			if verbose {
				fmt.Println("LLM produced:")
				fmt.Println(resp)
			}

			// Trimming the header and footer in the response, generated by the LLM if any
			resp = utils.TrimResponse(resp)

			// ⑤ Decode into Go structs
			var tasks []tasklist.Task
			if err := json.Unmarshal([]byte(resp), &tasks); err != nil {
				return fmt.Errorf("unmarshaling tasks: %w", err)
			}

			// ── Confirmation & fine‑tune loop ─────────────────────────────────────────
			confirmed := false
			for !confirmed {
				// Show tasks to the user
				tasksJSON, _ := json.MarshalIndent(tasks, "", "  ")
				fmt.Println("Generated tasks:")
				fmt.Println(string(tasksJSON))

				// Ask for confirmation
				fmt.Print("Confirm adding these tasks? (yes/no). If no, please include how to make it better: ")
				reader := bufio.NewReader(os.Stdin)
				answer, _ := reader.ReadString('\n')
				answer = strings.TrimSpace(strings.ToLower(answer))

				if answer == "yes" || answer == "y" {
					confirmed = true
					break
				}

				// User said no – ask LLM to refine tasks based on the original note
				finePrompt := fmt.Sprintf(`User declined the generated tasks.

Original note: "%s"

User feedback: "%s"

With user feedback, please revise the task list to better reflect the note. Return a JSON array of tasks with "content" and "done" only, no extra keys, no explaination`, rawInput, answer)
				fineResp, err := llmAgent.Prompt(finePrompt)
				if err != nil {
					return fmt.Errorf("LLM refine prompt error: %w", err)
				}
				fineOutput, err := fineResp.GetResponse()
				if err != nil {
					return fmt.Errorf("reading LLM refine response: %w", err)
				}
				fineOutput = utils.TrimResponse(fineOutput)

				if verbose {
					fmt.Println("LLM refinement:")
					fmt.Println(fineOutput)
				}

				if err := json.Unmarshal([]byte(fineOutput), &tasks); err != nil {
					return fmt.Errorf("unmarshaling refined tasks: %w", err)
				}
			}
			// ────────────────────────────────────────────────────────────────────────

			// ⑥ Append each new task to the master list
			master := GetTaskList()
			for _, t := range tasks {
				master.AddTask(&t)
			}

			// ⑦ Persist the updated list
			master.Save()

			fmt.Printf("✅ Added %d task(s) to the list.\n", len(tasks))
			printToStdout()
			return nil
		},
	}
}
