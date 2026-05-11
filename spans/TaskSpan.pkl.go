// Code generated from Pkl module `tempo.pipeline.TaskSpans`. DO NOT EDIT.
package spans

type TaskSpan struct {
	GoName string `pkl:"goName" json:"goName,omitempty"`

	Name string `pkl:"name" json:"name,omitempty"`

	Uuid string `pkl:"uuid" json:"uuid,omitempty"`

	TraceID string `pkl:"traceID" json:"traceID,omitempty"`

	SpanID string `pkl:"spanID" json:"spanID,omitempty"`

	Required bool `pkl:"required" json:"required,omitempty"`

	Status *string `pkl:"status" json:"status,omitempty"`

	Project *string `pkl:"project" json:"project,omitempty"`

	Tags []string `pkl:"tags" json:"tags,omitempty"`

	Depends []SpanRef `pkl:"depends" json:"depends,omitempty"`
}
