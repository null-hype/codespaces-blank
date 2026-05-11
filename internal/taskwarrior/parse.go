package taskwarrior

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Task struct {
	Description string   `json:"description"`
	Depends     []string `json:"depends,omitempty"`
	Project     string   `json:"project,omitempty"`
	Status      string   `json:"status,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	UUID        string   `json:"uuid"`
}

func ParseExport(contents string) ([]Task, error) {
	trimmed := strings.TrimSpace(contents)
	if trimmed == "" {
		return nil, fmt.Errorf("task export is empty")
	}

	var tasks []Task
	if strings.HasPrefix(trimmed, "[") {
		if err := json.Unmarshal([]byte(trimmed), &tasks); err != nil {
			return nil, fmt.Errorf("parse task export JSON array: %w", err)
		}
	} else {
		for _, line := range strings.Split(trimmed, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || !strings.HasPrefix(line, "{") {
				continue
			}
			var task Task
			if err := json.Unmarshal([]byte(line), &task); err != nil {
				return nil, fmt.Errorf("parse task export JSONL line %q: %w", line, err)
			}
			tasks = append(tasks, task)
		}
	}
	if len(tasks) == 0 {
		return nil, fmt.Errorf("task export contained no task records")
	}
	return tasks, nil
}
