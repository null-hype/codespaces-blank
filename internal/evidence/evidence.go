package evidence

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"dagger/tempo-pipeline/domain"
	"dagger/tempo-pipeline/projections"
)

type Event struct {
	Time        string `json:"time"`
	PlanID      string `json:"plan_id"`
	TaskID      string `json:"task_id"`
	Task        string `json:"task"`
	Kind        string `json:"kind"`
	Command     string `json:"command,omitempty"`
	Description string `json:"description"`
}

var recorder = struct {
	sync.Mutex
	events []Event
}{}

func Reset() {
	recorder.Lock()
	defer recorder.Unlock()
	recorder.events = nil
}

func RecordTask(planID string, task domain.Task, kind string) string {
	command := ""
	if task.Command != nil {
		command = task.Command.Name
	}
	event := Event{
		Time:        time.Now().UTC().Format(time.RFC3339Nano),
		PlanID:      planID,
		TaskID:      task.Id,
		Task:        task.Description,
		Kind:        kind,
		Command:     command,
		Description: "required task evidence emitted",
	}
	recorder.Lock()
	recorder.events = append(recorder.events, event)
	recorder.Unlock()

	encoded, err := json.Marshal(event)
	if err != nil {
		return fmt.Sprintf(`{"plan_id":%q,"task_id":%q,"kind":%q,"error":%q}`, planID, task.Id, kind, err.Error())
	}
	return string(encoded)
}

func Snapshot() []Event {
	recorder.Lock()
	defer recorder.Unlock()
	return append([]Event(nil), recorder.events...)
}

func Verify(expectations []projections.EvidenceExpectation) error {
	events := Snapshot()
	for _, expectation := range expectations {
		if !expectation.Required || expectation.Kind == "otel-span" || expectation.Kind == "diagnostic" {
			continue
		}
		found := false
		for _, event := range events {
			if event.TaskID == expectation.TaskID && event.Kind == expectation.Kind {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("missing required %s evidence for %s", expectation.Kind, expectation.TaskID)
		}
	}
	return nil
}
