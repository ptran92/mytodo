package commands

import (
	"fmt"
	"io"
	"mytodo/lib/tasklist"
	"os"
	"strconv"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	MasterTasks *tasklist.TaskList
)

func SetMasterTasks(t *tasklist.TaskList) {
	MasterTasks = t
}

func nicePrint(writer io.Writer, tasks []tasklist.Task) error {
	// Define styled printers
	success := color.New(color.FgGreen, color.Bold).Sprintf
	info := color.New(color.FgCyan).Sprintf

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

	addCmd := &cobra.Command{
		Use:   "add [task]",
		Short: "Add a new task",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			defer printToStdout()
			todo := args[0]
			if verbose {
				fmt.Println("Adding task:", todo)
			}

			task := tasklist.Task{
				Content: todo,
				Done:    false,
			}

			GetTaskList().AddTask(&task)

		},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all tasks",
		Run: func(cmd *cobra.Command, args []string) {
			if GetTaskList().NumberOfTasks() == 0 {
				fmt.Println("No tasks found.")
				return
			}

			printToStdout()
		},
	}

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

	rootCmd.AddCommand(
		addCmd,
		listCmd,
		removeCommand,
		doneCommand,
		undoneCommand,
		editCommand,
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
