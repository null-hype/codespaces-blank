package compiler

import (
	"testing"

	"dagger/tempo-pipeline/domain"
	"dagger/tempo-pipeline/internal/taskwarrior"
)

func TestGenerateValidDefaultPlan(t *testing.T) {
	result := Generate(defaultTasks(), domain.Current)

	if HasErrors(result) {
		t.Fatalf("expected no errors, got %#v", result.Diagnostics)
	}
	if len(result.Nodes) != 3 {
		t.Fatalf("nodes = %d, want 3", len(result.Nodes))
	}
	if got, want := result.TopologicalOrder, []string{"task.run-tempo", "task.tempo", "task.check"}; !sameStringsOrdered(got, want) {
		t.Fatalf("topological order = %v, want %v", got, want)
	}
	if len(result.Evidence) != 5 {
		t.Fatalf("evidence entries = %d, want 5", len(result.Evidence))
	}
	if len(result.OtelProjection.Spans) != 2 {
		t.Fatalf("otel spans = %d, want 2", len(result.OtelProjection.Spans))
	}
	artifacts, err := RenderArtifacts(result)
	if err != nil {
		t.Fatal(err)
	}
	if artifacts.DAGJSON == "" || artifacts.DiagnosticsJSON == "" || artifacts.NormalizedTaskwarriorJSONL == "" || artifacts.EvidenceContractJSON == "" || artifacts.OtelProjectionJSON == "" || artifacts.RunbookMarkdown == "" {
		t.Fatalf("expected every artifact to be populated: %#v", artifacts)
	}
}

func TestGenerateReportsMissingBinding(t *testing.T) {
	tasks := append(defaultTasks(), taskwarrior.Task{UUID: "44444444-4444-4444-8444-444444444444", Description: "unbound", Project: "tempo-pipeline"})
	result := Generate(tasks, domain.Current)

	assertDiagnostic(t, result, "MISSING_BINDING")
}

func TestGenerateReportsUnknownDependency(t *testing.T) {
	tasks := defaultTasks()
	tasks[1].Depends = []string{"missing"}
	result := Generate(tasks, domain.Current)

	assertDiagnostic(t, result, "UNKNOWN_DEPENDENCY")
}

func TestGenerateReportsCycle(t *testing.T) {
	tasks := defaultTasks()
	tasks[0].Depends = []string{tasks[2].UUID}
	result := Generate(tasks, domain.Current)

	assertDiagnostic(t, result, "CYCLE")
}

func TestGenerateReportsDuplicateUUID(t *testing.T) {
	tasks := defaultTasks()
	tasks[1].UUID = tasks[0].UUID
	result := Generate(tasks, domain.Current)

	assertDiagnostic(t, result, "DUPLICATE_UUID")
}

func TestGenerateReportsDisallowedCapability(t *testing.T) {
	cfg := domain.Current
	cfg.BindingRules = append([]domain.TaskBindingRule(nil), cfg.BindingRules...)
	cfg.BindingRules[0].Actor = "actor.agent"
	result := Generate(defaultTasks(), cfg)

	assertDiagnostic(t, result, "DISALLOWED_CAPABILITY")
}

func defaultTasks() []taskwarrior.Task {
	return []taskwarrior.Task{
		{UUID: "11111111-1111-4111-8111-111111111111", Description: "run-tempo", Project: "tempo-pipeline", Status: "pending", Tags: []string{"dag", "dagger"}},
		{UUID: "22222222-2222-4222-8222-222222222222", Description: "tempo", Project: "tempo-pipeline", Status: "pending", Tags: []string{"dag", "dagger"}, Depends: []string{"11111111-1111-4111-8111-111111111111"}},
		{UUID: "33333333-3333-4333-8333-333333333333", Description: "check", Project: "tempo-pipeline", Status: "pending", Tags: []string{"dag", "dagger"}, Depends: []string{"22222222-2222-4222-8222-222222222222"}},
	}
}

func assertDiagnostic(t *testing.T, result Result, code string) {
	t.Helper()
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == code {
			return
		}
	}
	t.Fatalf("missing diagnostic %s in %#v", code, result.Diagnostics)
}

func sameStringsOrdered(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
