// Code generated from pkl/taskSpans.pkl. DO NOT EDIT.
package spans

var (
	TraceID = "8ecc483576007b7d4f49decdd2ffb495"

	RunTempo = TaskSpan{
		GoName:   "RunTempo",
		Name:     "run-tempo",
		Uuid:     "11111111-1111-4111-8111-111111111111",
		TraceID:  "8ecc483576007b7d4f49decdd2ffb495",
		SpanID:   "a0b94f4b868e52c2",
		Required: true,
		Status:   ptr("pending"),
		Project:  ptr("tempo-pipeline"),
		Tags:     []string{"dag", "dagger"},
	}

	Tempo = TaskSpan{
		GoName:   "Tempo",
		Name:     "tempo",
		Uuid:     "22222222-2222-4222-8222-222222222222",
		TraceID:  "8ecc483576007b7d4f49decdd2ffb495",
		SpanID:   "335adf755a3b6db2",
		Required: true,
		Status:   ptr("pending"),
		Project:  ptr("tempo-pipeline"),
		Tags:     []string{"dag", "dagger"},
		Depends: []SpanRef{
			{
				Name:   "run-tempo",
				Uuid:   "11111111-1111-4111-8111-111111111111",
				SpanID: "a0b94f4b868e52c2",
			},
		},
	}

	Check = TaskSpan{
		GoName:   "Check",
		Name:     "check",
		Uuid:     "33333333-3333-4333-8333-333333333333",
		TraceID:  "8ecc483576007b7d4f49decdd2ffb495",
		SpanID:   "917039eafa6c0817",
		Required: false,
		Status:   ptr("pending"),
		Project:  ptr("tempo-pipeline"),
		Tags:     []string{"dag", "dagger"},
		Depends: []SpanRef{
			{
				Name:   "tempo",
				Uuid:   "22222222-2222-4222-8222-222222222222",
				SpanID: "335adf755a3b6db2",
			},
		},
	}

	Expected = []TaskSpan{RunTempo, Tempo}
	All      = []TaskSpan{RunTempo, Tempo, Check}
)

func ptr(value string) *string {
	return &value
}
