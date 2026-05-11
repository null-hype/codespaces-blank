// Code generated from Pkl module `tempo.pipeline.domain.WorkDomain`. DO NOT EDIT.
package domain

type TaskBindingRule struct {
	Id string `pkl:"id" json:"id,omitempty"`

	TaskID *string `pkl:"taskID" json:"taskID,omitempty"`

	Match TaskMatch `pkl:"match" json:"match,omitempty"`

	Actor string `pkl:"actor" json:"actor,omitempty"`

	Capability string `pkl:"capability" json:"capability,omitempty"`

	Command *Command `pkl:"command" json:"command,omitempty"`

	Evidence []EvidenceRequirement `pkl:"evidence" json:"evidence,omitempty"`
}
