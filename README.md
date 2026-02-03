# MyTodo

A powerful command-line TODO application built with Go that combines traditional task management with AI-powered features.

## Features

- **Simple Task Management**: Add, list, edit, remove, and mark tasks as done/undone
- **AI Integration**: Supports both OpenAI and Ollama for intelligent task parsing and formatting
- **Natural Language Input**: Use AI to convert free-form text into structured tasks
- **Task Comments**: Add notes and comments to any task
- **Persistent Storage**: Tasks are automatically saved to `~/.mytodo.json`
- **Beautiful Output**: Colored terminal output with status icons
- **Interactive Confirmation**: Review AI-generated tasks before adding them

## Installation

### Prerequisites

- Go 1.22.4 or higher
- (Optional) OpenAI API key or Ollama installation for AI features

### Build from Source

```bash
git clone <repository-url>
cd mytodo
go build -o mytodo ./cmd
```

Move the binary to your PATH:

```bash
sudo mv mytodo /usr/local/bin/
```

## Configuration

### AI Backend Selection

Edit `cmd/cmd.go` to choose your AI backend:

```go
const (
    SelectedAgentBackend = OpenAIAgent   // or OllamaAIAgent
)
```

### Environment Variables

**For OpenAI:**
```bash
export OPEN_AI_API_KEY="your-api-key-here"
```

**For Ollama:**
```bash
export USE_AI="true"
# Ensure Ollama is running on localhost:11434
```

**Disable AI Features:**
Simply don't set any AI-related environment variables, and the app will work in traditional mode.

## Usage

### Basic Commands

#### Add a Task

**Without AI:**
```bash
mytodo add "Buy groceries"
```

**With AI (Natural Language):**
```bash
mytodo add "I need to finish the report by Friday, call mom, and pick up groceries"
```

The AI will parse your input and create multiple structured tasks. You'll see a confirmation prompt where you can approve or refine the tasks.

**From stdin:**
```bash
echo "Complete project documentation and deploy to production" | mytodo add
```

#### List All Tasks

**Basic listing:**
```bash
mytodo list
```

**With AI summary:**
```bash
mytodo list --summary
# or
mytodo list -s
```

#### Mark Task as Done

```bash
mytodo done 0
```

#### Mark Task as Not Done

```bash
mytodo undone 0
```

#### Edit a Task

```bash
mytodo edit 0 "Updated task description"
```

#### Add a Comment to a Task

```bash
mytodo cm 0 "This is a comment on the first task"
```

#### Remove a Task

```bash
mytodo remove 0
```

### Verbose Mode

Add `-v` or `--verbose` flag to any command for detailed output:

```bash
mytodo -v add "New task"
mytodo -v list
```

## AI Features

### Natural Language Task Creation

When AI is enabled, the `add` command can parse complex sentences into multiple tasks:

**Input:**
```bash
mytodo add "Tomorrow I need to finish the presentation, send emails to the team, and schedule a meeting with John"
```

**Output:**
The AI will generate:
- Task 1: "Finish the presentation" (done: false)
- Task 2: "Send emails to the team" (done: false)
- Task 3: "Schedule a meeting with John" (done: false)

You'll be prompted to confirm or provide feedback for refinement.

### Interactive Refinement

If you're not satisfied with the generated tasks:

```bash
Confirm adding these tasks? (yes/no). If no, please include how to make it better:
> no, make the first task more specific about what needs to be in the presentation
```

The AI will regenerate the tasks based on your feedback.

### AI-Formatted Output

With AI enabled, the `list` command presents tasks in a beautifully formatted way with emojis and proper indentation for comments.

### Summary Generation

Use the `--summary` flag to get a concise overview:

```bash
mytodo list --summary
```

Output example:
```
[Task list displayed]

Summary: You have 5 pending tasks focused on project completion and team coordination, with 2 already completed.
```

## Project Structure

```
mytodo/
├── cmd/
│   └── cmd.go              # Main entry point
├── lib/
│   ├── agent/
│   │   └── agent.go        # LLM agent implementations (OpenAI, Ollama)
│   ├── commands/
│   │   └── commands.go     # CLI command definitions
│   ├── tasklist/
│   │   └── tasklist.go     # Task data structures and persistence
│   └── utils/
│       └── utils.go        # Utility functions
├── go.mod
└── go.sum
```

## Data Storage

Tasks are stored in JSON format at `~/.mytodo.json`:

```json
{
  "tasks": [
    {
      "content": "Buy groceries",
      "done": false,
      "comments": ["Need milk and eggs"]
    }
  ]
}
```

## AI Agent Details

### OpenAI Agent

- Uses GPT-4.1 model
- Max output tokens: 4096
- Temperature: 0.7
- Requires `OPEN_AI_API_KEY` environment variable

### Ollama Agent

- Uses `gpt-oss:20b` model
- Connects to `http://localhost:11434`
- Requires Ollama to be running locally

## Examples

### Complete Workflow

```bash
# Add tasks using natural language
mytodo add "This week: review PRs, update documentation, fix bug #123"

# List all tasks with a summary
mytodo list --summary

# Add a comment to track progress
mytodo cm 0 "Started working on this, found additional edge cases"

# Mark completed tasks as done
mytodo done 0

# Edit a task for clarity
mytodo edit 1 "Update API documentation with new endpoints"

# Remove unnecessary tasks
mytodo remove 2

# View final status
mytodo list
```

## Dependencies

- [cobra](https://github.com/spf13/cobra) - CLI framework
- [color](https://github.com/fatih/color) - Terminal color output
- Standard Go libraries for HTTP, JSON, and file I/O

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

[Add your license here]

## Troubleshooting

### OpenAI API Errors

If you see truncated responses, ensure your `max_output_tokens` is set to at least 4096 in `lib/agent/agent.go`.

### Ollama Connection Issues

Ensure Ollama is running:
```bash
ollama serve
```

### Task File Permissions

If you encounter permission errors, check that `~/.mytodo.json` is readable and writable:
```bash
chmod 644 ~/.mytodo.json
```
