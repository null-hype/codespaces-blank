// Code generated from Pkl module `tempo.pipeline.domain.WorkDomain`. DO NOT EDIT.
package domain

type Task struct {
	Id string `pkl:"id" json:"id,omitempty"`

	TaskwarriorUuid string `pkl:"taskwarriorUuid" json:"taskwarriorUuid,omitempty"`

	Description string `pkl:"description" json:"description,omitempty"`

	Project string `pkl:"project" json:"project,omitempty"`

	Status string `pkl:"status" json:"status,omitempty"`

	Tags []string `pkl:"tags" json:"tags,omitempty"`

	DependsOn []string `pkl:"dependsOn" json:"dependsOn,omitempty"`

	Actor string `pkl:"actor" json:"actor,omitempty"`

	Capability string `pkl:"capability" json:"capability,omitempty"`

	Command *Command `pkl:"command" json:"command,omitempty"`

	Evidence []EvidenceRequirement `pkl:"evidence" json:"evidence,omitempty"`
}
