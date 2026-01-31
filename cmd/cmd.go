package main

import (
	"fmt"
	"mytodo/lib/agent"
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
	llmAgent := agent.CreateLlmAgentDefault()

	commands.SetAgent(llmAgent)

	err := commands.PrepareCommands().Execute()
	if err != nil {
		fmt.Println("Error:", err)
	}

	// // Example usage
	// response, err := llmAgent.Prompt("Hello, Ollama! What is the capital of France?")
	// if err != nil {
	// 	fmt.Println("Error:", err)
	// 	return
	// }

	// text, err := response.GetResponse()
	// if err != nil {
	// 	fmt.Println("Error parsing JSON response:", err)
	// 	return
	// }
	// fmt.Println("LLM Response:")
	// fmt.Println(text)

}
