package main

import (
	"fmt"
	"mytodo/lib/commands"
	"mytodo/lib/tasklist"
	"os"
	"path"
)

var (
	MasterTasks *tasklist.TaskList
)

const (
	TrackFile = ".mytodo.json"
)

func init() {
	homePath := os.Getenv("HOME")
	if homePath == "" {
		homePath = "."
	}

	t := tasklist.NewTaskList(path.Join(homePath, TrackFile))
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
