package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	Tasks       []string
	MasterTasks *TaskList
)

const (
	PersistentFile = ".mytodo.json"
)

type Task struct {
	Content string `json:"content"`
	Done    bool   `json:"done"`
}

type TaskList struct {
	Tasks    []Task `json:"tasks"`
	filePath string `json:"-"`
}

func (t *TaskList) Save() {
	content, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling tasks:", err)
		return
	}
	os.WriteFile(PersistentFile, content, 0644)
}

func (t *TaskList) Load() error {
	content, err := os.ReadFile(PersistentFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		fmt.Println("Error reading tasks:", err)
		return err
	}
	return json.Unmarshal(content, &t)
}

func (t *TaskList) AddTask(task *Task) {
	t.Tasks = append(t.Tasks, *task)
	t.Save()
}

func (t *TaskList) RemoveTask(index int) {
	if index < 0 || index >= len(t.Tasks) {
		return
	}

	t.Tasks = append(t.Tasks[:index], t.Tasks[index+1:]...)
	t.Save()
}

func (t *TaskList) GetTask(index int) *Task {
	if index < 0 || index >= len(t.Tasks) {
		return nil
	}

	copy := t.Tasks[index]
	return &copy
}

func (t *TaskList) ReplaceTask(index int, newTask *Task) {
	if index < 0 || index >= len(t.Tasks) {
		return
	}

	t.Tasks[index] = *newTask
	t.Save()
}

func (t *TaskList) GetAllTasks() []Task {
	copy := make([]Task, 0, len(t.Tasks))

	copy = append(copy, t.Tasks[0:]...)
	return copy
}

func NicePrint(writer io.Writer, tasks []Task) error {
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

func GetTaskList() *TaskList {
	return MasterTasks
}

func prepareCommands() *cobra.Command {
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
			todo := args[0]
			if verbose {
				fmt.Println("Adding task:", todo)
			}

			task := Task{
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
			if len(MasterTasks.Tasks) == 0 {
				fmt.Println("No tasks found.")
				return
			}

			tasks := GetTaskList().GetAllTasks()

			NicePrint(os.Stdout, tasks)
		},
	}

	removeCommand := &cobra.Command{
		Use:   "remove [task number]",
		Short: "Remove a task by its number",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(MasterTasks.Tasks) == 0 {
				fmt.Println("No tasks to remove.")
				return
			}

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

func init() {
	t := &TaskList{Tasks: make([]Task, 0), filePath: PersistentFile}
	MasterTasks = t
	err := GetTaskList().Load()
	if err != nil {
		panic(fmt.Sprintf("Error loading tasks: %v", err))
	}
}

func main() {
	err := prepareCommands().Execute()
	if err != nil {
		fmt.Println("Error:", err)
	}
}
