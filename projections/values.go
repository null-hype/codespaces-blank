// Code generated from pkl/plans/tempo-pipeline-*.pkl. DO NOT EDIT.
package projections

var TraceID = "8ecc483576007b7d4f49decdd2ffb495"

var RunTempo = OtelSpan{
	TaskID:          "task.run-tempo",
	GoName:          "RunTempo",
	Name:            "run-tempo",
	TaskwarriorUuid: "11111111-1111-4111-8111-111111111111",
	TraceID:         "8ecc483576007b7d4f49decdd2ffb495",
	SpanID:          "a0b94f4b868e52c2",
	Required:        true,
	Project:         "project.tempo-pipeline",
	Status:          "pending",
	Tags:            []string{"dag", "dagger"},
}

var Tempo = OtelSpan{
	TaskID:          "task.tempo",
	GoName:          "Tempo",
	Name:            "tempo",
	TaskwarriorUuid: "22222222-2222-4222-8222-222222222222",
	TraceID:         "8ecc483576007b7d4f49decdd2ffb495",
	SpanID:          "335adf755a3b6db2",
	Required:        true,
	Project:         "project.tempo-pipeline",
	Status:          "pending",
	Tags:            []string{"dag", "dagger"},
	Depends: []SpanRef{
		{
			TaskID: "task.run-tempo",
			Name:   "run-tempo",
			SpanID: "a0b94f4b868e52c2",
		},
	},
}

var ExpectedOtel = []OtelSpan{RunTempo, Tempo}

var ExpectedEvidence = []EvidenceExpectation{
	{TaskID: "task.run-tempo", Kind: "jsonl-event", Required: true},
	{TaskID: "task.run-tempo", Kind: "otel-span", Required: true},
	{TaskID: "task.tempo", Kind: "jsonl-event", Required: true},
	{TaskID: "task.tempo", Kind: "otel-span", Required: true},
	{TaskID: "task.check", Kind: "diagnostic", Required: true},
}
