// Code generated from Pkl module `tempo.pipeline.domain.WorkDomain`. DO NOT EDIT.
package domain

type Scope struct {
	Id string `pkl:"id" json:"id,omitempty"`

	Description string `pkl:"description" json:"description,omitempty"`

	AllowedTargets []string `pkl:"allowedTargets" json:"allowedTargets,omitempty"`

	ForbiddenActions []string `pkl:"forbiddenActions" json:"forbiddenActions,omitempty"`
}
