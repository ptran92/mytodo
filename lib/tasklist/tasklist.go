package tasklist

import (
	"encoding/json"
	"fmt"
	"os"
)

type TaskList struct {
	Tasks    []Task `json:"tasks"`
	filePath string `json:"-"`
}

type Task struct {
	Content  string   `json:"content"`
	Done     bool     `json:"done"`
	Comments []string `json:"comments,omitempty"`
}

func NewTaskList(filepath string) *TaskList {
	return &TaskList{
		Tasks:    []Task{},
		filePath: filepath,
	}
}

func (t *TaskList) Save() {
	content, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling tasks:", err)
		return
	}
	os.WriteFile(t.filePath, content, 0644)
}

func (t *TaskList) Load() error {
	content, err := os.ReadFile(t.filePath)
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

func (t *TaskList) NumberOfTasks() int {
	return len(t.Tasks)
}

func (t *TaskList) GetAllTasks() []Task {
	if len(t.Tasks) == 0 {
		return nil
	}

	copy := make([]Task, 0, len(t.Tasks))

	copy = append(copy, t.Tasks[0:]...)
	return copy
}

func (t *TaskList) AddComment(index int, comment string) {
	if index < 0 || index >= len(t.Tasks) {
		return
	}

	t.Tasks[index].Comments = append(t.Tasks[index].Comments, comment)
	t.Save()
}
func (t *TaskList) GetComments(index int) []string {
	if index < 0 || index >= len(t.Tasks) {
		return nil
	}
	copies := make([]string, 0, len(t.Tasks[index].Comments))
	copies = append(copies, t.Tasks[index].Comments[0:]...)

	return copies
}
