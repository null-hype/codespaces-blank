// Code generated from Pkl module `tempo.pipeline.projections.Projections`. DO NOT EDIT.
package projections

import (
	"context"

	"github.com/apple/pkl-go/pkl"
)

type Projections struct {
	PlanID string `pkl:"planID" json:"planID,omitempty"`

	TraceID string `pkl:"traceID" json:"traceID,omitempty"`

	OtelSpans []OtelSpan `pkl:"otelSpans" json:"otelSpans,omitempty"`

	ExpectedOtel []OtelSpan `pkl:"expectedOtel" json:"expectedOtel,omitempty"`

	Evidence []EvidenceExpectation `pkl:"evidence" json:"evidence,omitempty"`

	ExpectedEvidence []EvidenceExpectation `pkl:"expectedEvidence" json:"expectedEvidence,omitempty"`
}

// LoadFromPath loads the pkl module at the given path and evaluates it into a Projections
func LoadFromPath(ctx context.Context, path string) (ret Projections, err error) {
	evaluator, err := pkl.NewEvaluator(ctx, pkl.PreconfiguredOptions)
	if err != nil {
		return ret, err
	}
	defer func() {
		cerr := evaluator.Close()
		if err == nil {
			err = cerr
		}
	}()
	ret, err = Load(ctx, evaluator, pkl.FileSource(path))
	return ret, err
}

// Load loads the pkl module at the given source and evaluates it with the given evaluator into a Projections
func Load(ctx context.Context, evaluator pkl.Evaluator, source *pkl.ModuleSource) (Projections, error) {
	var ret Projections
	err := evaluator.EvaluateModule(ctx, source, &ret)
	return ret, err
}
