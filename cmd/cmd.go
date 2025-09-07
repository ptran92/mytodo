package main

import (
	"fmt"
	"mytodo/lib/commands"
	"mytodo/lib/tasklist"
)

var (
	MasterTasks *tasklist.TaskList
)

const (
	PersistentFile = ".mytodo.json"
)

func init() {
	t := tasklist.NewTaskList(PersistentFile)
	commands.SetMasterTasks(t)
	err := commands.GetTaskList().Load()
	if err != nil {
		panic(fmt.Sprintf("Error loading tasks: %v", err))
	}
}

func main() {
	err := commands.PrepareCommands().Execute()
	if err != nil {
		fmt.Println("Error:", err)
	}
}
