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

type AgentBackend string

const (
	OpenAIAgent   AgentBackend = "OpenAI"
	OllamaAIAgent AgentBackend = "Ollama"
)

const (
	TrackFile            = ".mytodo.json"
	SelectedAgentBackend = OpenAIAgent
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
	var llmAgent agent.LlmAgent
	var err error

	if SelectedAgentBackend == OllamaAIAgent {
		llmAgent = agent.CreateLlmAgentDefault()

	} else if SelectedAgentBackend == OpenAIAgent {
		llmAgent, err = agent.CreateOpenAIAgentDefault()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	commands.SetAgent(llmAgent)
	err = commands.PrepareCommands().Execute()
	if err != nil {
		fmt.Println("Error:", err)
	}
}
