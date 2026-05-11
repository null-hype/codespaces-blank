// Code generated from pkl/plans/tempo-pipeline.pkl. DO NOT EDIT.
package domain

var Current = WorkDomain{
	Projects: []Project{
		{
			Id:   "project.tempo-pipeline",
			Name: "Tempo Pipeline",
			Kind: "toolchain",
		},
	},
	Actors: []Actor{
		{
			Id:                  "actor.agent",
			Kind:                "llm-agent",
			Description:         "Plans work as Taskwarrior tasks and updates the Dagger module.",
			AllowedCapabilities: []string{"cap.plan"},
		},
		{
			Id:                  "actor.dagger",
			Kind:                "dagger-function",
			Description:         "Runs the Dagger functions and emits execution evidence.",
			AllowedCapabilities: []string{"cap.dagger-service", "cap.evidence-check"},
		},
	},
	Capabilities: []Capability{
		{Id: "cap.plan", Kind: "plan", Tool: "taskwarrior", MaxRisk: "low"},
		{Id: "cap.dagger-service", Kind: "dagger-service", Tool: "dagger", MaxRisk: "low"},
		{Id: "cap.evidence-check", Kind: "evidence-check", Tool: "dagger-check", MaxRisk: "low"},
	},
	Scopes: []Scope{
		{
			Id:             "scope.local",
			Description:    "Local Dagger module and Tempo smoke-test environment.",
			AllowedTargets: []string{"dagger-module", "tempo-service"},
		},
	},
	BindingRules: []TaskBindingRule{
		{
			Id:         "bind.run-tempo",
			TaskID:     stringPtr("task.run-tempo"),
			Match:      TaskMatch{Description: stringPtr("run-tempo"), Project: stringPtr("tempo-pipeline"), Tags: []string{"dag", "dagger"}},
			Actor:      "actor.dagger",
			Capability: "cap.dagger-service",
			Command:    &Command{Kind: "dagger-function", Name: "RunTempo"},
			Evidence:   []EvidenceRequirement{{Id: "ev.run-tempo.jsonl", Kind: "jsonl-event", Required: true}, {Id: "ev.run-tempo.otel", Kind: "otel-span", Required: true}},
		},
		{
			Id:         "bind.tempo",
			TaskID:     stringPtr("task.tempo"),
			Match:      TaskMatch{Description: stringPtr("tempo"), Project: stringPtr("tempo-pipeline"), Tags: []string{"dag", "dagger"}},
			Actor:      "actor.dagger",
			Capability: "cap.dagger-service",
			Command:    &Command{Kind: "dagger-function", Name: "Tempo"},
			Evidence:   []EvidenceRequirement{{Id: "ev.tempo.jsonl", Kind: "jsonl-event", Required: true}, {Id: "ev.tempo.otel", Kind: "otel-span", Required: true}},
		},
		{
			Id:         "bind.check",
			TaskID:     stringPtr("task.check"),
			Match:      TaskMatch{Description: stringPtr("check"), Project: stringPtr("tempo-pipeline"), Tags: []string{"dag", "dagger"}},
			Actor:      "actor.dagger",
			Capability: "cap.evidence-check",
			Command:    &Command{Kind: "dagger-function", Name: "Check"},
			Evidence:   []EvidenceRequirement{{Id: "ev.check.diagnostic", Kind: "diagnostic", Required: true}},
		},
	},
	Plans: []WorkPlan{TempoPipelinePlan},
}

var TempoPipelinePlan = WorkPlan{
	Id:      "plan.tempo-pipeline",
	Project: "project.tempo-pipeline",
	Scope:   "scope.local",
	Tasks: []Task{
		RunTempoTask,
		TempoTask,
		CheckTask,
	},
}

var RunTempoTask = Task{
	Id:              "task.run-tempo",
	TaskwarriorUuid: "11111111-1111-4111-8111-111111111111",
	Description:     "run-tempo",
	Project:         "project.tempo-pipeline",
	Status:          "pending",
	Tags:            []string{"dag", "dagger"},
	Actor:           "actor.dagger",
	Capability:      "cap.dagger-service",
	Command:         &Command{Kind: "dagger-function", Name: "RunTempo"},
	Evidence:        []EvidenceRequirement{{Id: "ev.run-tempo.jsonl", Kind: "jsonl-event", Required: true}, {Id: "ev.run-tempo.otel", Kind: "otel-span", Required: true}},
}

var TempoTask = Task{
	Id:              "task.tempo",
	TaskwarriorUuid: "22222222-2222-4222-8222-222222222222",
	Description:     "tempo",
	Project:         "project.tempo-pipeline",
	Status:          "pending",
	Tags:            []string{"dag", "dagger"},
	DependsOn:       []string{"task.run-tempo"},
	Actor:           "actor.dagger",
	Capability:      "cap.dagger-service",
	Command:         &Command{Kind: "dagger-function", Name: "Tempo"},
	Evidence:        []EvidenceRequirement{{Id: "ev.tempo.jsonl", Kind: "jsonl-event", Required: true}, {Id: "ev.tempo.otel", Kind: "otel-span", Required: true}},
}

var CheckTask = Task{
	Id:              "task.check",
	TaskwarriorUuid: "33333333-3333-4333-8333-333333333333",
	Description:     "check",
	Project:         "project.tempo-pipeline",
	Status:          "pending",
	Tags:            []string{"dag", "dagger"},
	DependsOn:       []string{"task.tempo"},
	Actor:           "actor.dagger",
	Capability:      "cap.evidence-check",
	Command:         &Command{Kind: "dagger-function", Name: "Check"},
	Evidence:        []EvidenceRequirement{{Id: "ev.check.diagnostic", Kind: "diagnostic", Required: true}},
}

func stringPtr(value string) *string {
	return &value
}
