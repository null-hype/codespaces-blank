package planner

import (
	"fmt"
	"strings"

	"dagger/tempo-pipeline/domain"
	"dagger/tempo-pipeline/internal/compiler"
	"dagger/tempo-pipeline/internal/taskwarrior"
)

func CompileTaskExport(tasks []taskwarrior.Task, cfg domain.WorkDomain) compiler.Result {
	return compiler.Generate(tasks, cfg)
}

func ValidateTaskExport(tasks []taskwarrior.Task, cfg domain.WorkDomain) (compiler.Result, error) {
	result := CompileTaskExport(tasks, cfg)
	if compiler.HasErrors(result) {
		return result, fmt.Errorf("task export compiled with failing diagnostics")
	}
	return result, nil
}

func ValidateCurrentPlanExport(contents string) (compiler.Result, error) {
	tasks, err := taskwarrior.ParseExport(contents)
	if err != nil {
		return compiler.Result{}, err
	}
	return ValidateTaskExport(tasks, domain.Current)
}

func Summary(result compiler.Result) string {
	taskNames := make([]string, 0, len(result.Nodes))
	for _, task := range result.Nodes {
		taskNames = append(taskNames, task.Description)
	}
	return fmt.Sprintf("ok: compiled %s with tasks %s", result.PlanID, strings.Join(taskNames, ", "))
}
