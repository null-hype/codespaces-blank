// Code generated from Pkl module `tempo.pipeline.TaskSpans`. DO NOT EDIT.
package spans

import (
	"context"

	"github.com/apple/pkl-go/pkl"
)

type TaskSpans struct {
	TraceID string `pkl:"traceID" json:"traceID,omitempty"`

	Spans []TaskSpan `pkl:"spans" json:"spans,omitempty"`

	DependenciesKnown bool `pkl:"dependenciesKnown" json:"dependenciesKnown,omitempty"`

	Expected []TaskSpan `pkl:"expected" json:"expected,omitempty"`
}

// LoadFromPath loads the pkl module at the given path and evaluates it into a TaskSpans
func LoadFromPath(ctx context.Context, path string) (ret TaskSpans, err error) {
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

// Load loads the pkl module at the given source and evaluates it with the given evaluator into a TaskSpans
func Load(ctx context.Context, evaluator pkl.Evaluator, source *pkl.ModuleSource) (TaskSpans, error) {
	var ret TaskSpans
	err := evaluator.EvaluateModule(ctx, source, &ret)
	return ret, err
}
