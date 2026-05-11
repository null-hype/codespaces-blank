// Code generated from Pkl module `tempo.pipeline.domain.WorkDomain`. DO NOT EDIT.
package domain

import (
	"context"

	"github.com/apple/pkl-go/pkl"
)

type WorkDomain struct {
	Projects []Project `pkl:"projects" json:"projects,omitempty"`

	Actors []Actor `pkl:"actors" json:"actors,omitempty"`

	Capabilities []Capability `pkl:"capabilities" json:"capabilities,omitempty"`

	Scopes []Scope `pkl:"scopes" json:"scopes,omitempty"`

	Plans []WorkPlan `pkl:"plans" json:"plans,omitempty"`

	ProjectRefsKnown bool `pkl:"projectRefsKnown" json:"projectRefsKnown,omitempty"`

	TaskRefsKnown bool `pkl:"taskRefsKnown" json:"taskRefsKnown,omitempty"`

	ActorCapabilitiesAllowed bool `pkl:"actorCapabilitiesAllowed" json:"actorCapabilitiesAllowed,omitempty"`
}

// LoadFromPath loads the pkl module at the given path and evaluates it into a WorkDomain
func LoadFromPath(ctx context.Context, path string) (ret WorkDomain, err error) {
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

// Load loads the pkl module at the given source and evaluates it with the given evaluator into a WorkDomain
func Load(ctx context.Context, evaluator pkl.Evaluator, source *pkl.ModuleSource) (WorkDomain, error) {
	var ret WorkDomain
	err := evaluator.EvaluateModule(ctx, source, &ret)
	return ret, err
}
