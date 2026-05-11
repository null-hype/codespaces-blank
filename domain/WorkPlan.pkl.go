// Code generated from Pkl module `tempo.pipeline.domain.WorkDomain`. DO NOT EDIT.
package domain

type WorkPlan struct {
	Id string `pkl:"id" json:"id,omitempty"`

	Project string `pkl:"project" json:"project,omitempty"`

	Scope string `pkl:"scope" json:"scope,omitempty"`

	Tasks []Task `pkl:"tasks" json:"tasks,omitempty"`
}
