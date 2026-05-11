package planner

import (
	"fmt"
	"sort"
	"strings"

	"dagger/tempo-pipeline/domain"
	"dagger/tempo-pipeline/internal/taskwarrior"
)

func ValidateTaskExport(tasks []taskwarrior.Task, plan domain.WorkPlan) error {
	byUUID := map[string]taskwarrior.Task{}
	for _, task := range tasks {
		if task.UUID == "" {
			return fmt.Errorf("task export contains a task with no uuid")
		}
		if _, exists := byUUID[task.UUID]; exists {
			return fmt.Errorf("task export contains duplicate uuid %s", task.UUID)
		}
		byUUID[task.UUID] = task
	}

	for _, planned := range plan.Tasks {
		task, ok := byUUID[planned.TaskwarriorUuid]
		if !ok {
			return fmt.Errorf("task export missing planned task %s (%s)", planned.Id, planned.TaskwarriorUuid)
		}
		if task.Description != planned.Description {
			return fmt.Errorf("task %s description mismatch: got %q, want %q", planned.Id, task.Description, planned.Description)
		}
		if !sameStrings(task.Tags, planned.Tags) {
			return fmt.Errorf("task %s tags mismatch: got %v, want %v", planned.Id, task.Tags, planned.Tags)
		}

		wantDeps := dependencyUUIDs(planned, plan)
		if !sameStrings(task.Depends, wantDeps) {
			return fmt.Errorf("task %s dependencies mismatch: got %v, want %v", planned.Id, task.Depends, wantDeps)
		}
	}
	return nil
}

func ValidateCurrentPlanExport(contents string) error {
	tasks, err := taskwarrior.ParseExport(contents)
	if err != nil {
		return err
	}
	return ValidateTaskExport(tasks, domain.TempoPipelinePlan)
}

func Summary(plan domain.WorkPlan) string {
	taskNames := make([]string, 0, len(plan.Tasks))
	for _, task := range plan.Tasks {
		taskNames = append(taskNames, task.Description)
	}
	return fmt.Sprintf("ok: validated %s with tasks %s", plan.Id, strings.Join(taskNames, ", "))
}

func dependencyUUIDs(task domain.Task, plan domain.WorkPlan) []string {
	uuids := make([]string, 0, len(task.DependsOn))
	for _, depID := range task.DependsOn {
		for _, candidate := range plan.Tasks {
			if candidate.Id == depID {
				uuids = append(uuids, candidate.TaskwarriorUuid)
				break
			}
		}
	}
	return uuids
}

func sameStrings(a []string, b []string) bool {
	ac := append([]string(nil), a...)
	bc := append([]string(nil), b...)
	sort.Strings(ac)
	sort.Strings(bc)
	if len(ac) != len(bc) {
		return false
	}
	for i := range ac {
		if ac[i] != bc[i] {
			return false
		}
	}
	return true
}
