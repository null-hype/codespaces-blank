// Code generated from Pkl module `tempo.pipeline.domain.WorkDomain`. DO NOT EDIT.
package domain

type Actor struct {
	Id string `pkl:"id" json:"id,omitempty"`

	Kind string `pkl:"kind" json:"kind,omitempty"`

	Description string `pkl:"description" json:"description,omitempty"`

	AllowedCapabilities []string `pkl:"allowedCapabilities" json:"allowedCapabilities,omitempty"`
}
