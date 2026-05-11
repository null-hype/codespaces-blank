// Code generated from Pkl module `tempo.pipeline.projections.Projections`. DO NOT EDIT.
package projections

type OtelSpan struct {
	TaskID string `pkl:"taskID" json:"taskID,omitempty"`

	GoName string `pkl:"goName" json:"goName,omitempty"`

	Name string `pkl:"name" json:"name,omitempty"`

	TaskwarriorUuid string `pkl:"taskwarriorUuid" json:"taskwarriorUuid,omitempty"`

	TraceID string `pkl:"traceID" json:"traceID,omitempty"`

	SpanID string `pkl:"spanID" json:"spanID,omitempty"`

	Required bool `pkl:"required" json:"required,omitempty"`

	Project string `pkl:"project" json:"project,omitempty"`

	Status string `pkl:"status" json:"status,omitempty"`

	Tags []string `pkl:"tags" json:"tags,omitempty"`

	Depends []SpanRef `pkl:"depends" json:"depends,omitempty"`
}
