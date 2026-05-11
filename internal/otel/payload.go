package otel

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"dagger/tempo-pipeline/projections"
	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	resourcev1 "go.opentelemetry.io/proto/otlp/resource/v1"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"
)

func PayloadBase64(span projections.OtelSpan) (string, error) {
	traceID, err := hex.DecodeString(span.TraceID)
	if err != nil {
		return "", fmt.Errorf("decode trace id for %s: %w", span.GoName, err)
	}
	spanID, err := hex.DecodeString(span.SpanID)
	if err != nil {
		return "", fmt.Errorf("decode span id for %s: %w", span.GoName, err)
	}
	now := time.Now().UnixNano()

	payload := &tracev1.TracesData{
		ResourceSpans: []*tracev1.ResourceSpans{
			{
				Resource: &resourcev1.Resource{
					Attributes: []*commonv1.KeyValue{
						kvString("service.name", "taskwarrior-dag"),
						kvString("dagger.module", "tempo-pipeline"),
					},
				},
				ScopeSpans: []*tracev1.ScopeSpans{
					{
						Scope: &commonv1.InstrumentationScope{
							Name:    "dagger/tempo-pipeline/" + span.GoName,
							Version: "pkl-domain-otel-projection",
						},
						Spans: []*tracev1.Span{
							{
								TraceId:           traceID,
								SpanId:            spanID,
								Name:              span.Name,
								Kind:              tracev1.Span_SPAN_KIND_INTERNAL,
								StartTimeUnixNano: uint64(now),
								EndTimeUnixNano:   uint64(now + 500_000),
								Attributes:        spanAttributes(span),
								Links:             spanLinks(span.TraceID, span.Depends),
								Status:            &tracev1.Status{},
							},
						},
					},
				},
			},
		},
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal OTLP payload for %s: %w", span.GoName, err)
	}
	return base64.StdEncoding.EncodeToString(payloadBytes), nil
}

func spanAttributes(span projections.OtelSpan) []*commonv1.KeyValue {
	attrs := []*commonv1.KeyValue{
		kvString("task.id", span.TaskID),
		kvString("task.uuid", span.TaskwarriorUuid),
		kvString("task.description", span.Name),
		kvString("task.go_name", span.GoName),
		kvString("task.emitter", "dagger-function"),
		kvString("task.trace.namespace", "tempo-pipeline/task-dag"),
		kvString("task.status", span.Status),
		kvString("task.project", span.Project),
	}
	if len(span.Tags) > 0 {
		attrs = append(attrs, kvStrings("task.tags", span.Tags))
	}
	if len(span.Depends) > 0 {
		deps := make([]string, 0, len(span.Depends))
		for _, dep := range span.Depends {
			deps = append(deps, dep.TaskID)
		}
		attrs = append(attrs, kvStrings("task.depends", deps))
	}
	return attrs
}

func spanLinks(traceID string, refs []projections.SpanRef) []*tracev1.Span_Link {
	links := make([]*tracev1.Span_Link, 0, len(refs))
	traceBytes, err := hex.DecodeString(traceID)
	if err != nil {
		return links
	}
	for _, ref := range refs {
		spanBytes, err := hex.DecodeString(ref.SpanID)
		if err != nil {
			continue
		}
		links = append(links, &tracev1.Span_Link{
			TraceId: traceBytes,
			SpanId:  spanBytes,
			Attributes: []*commonv1.KeyValue{
				kvString("task.dependency.id", ref.TaskID),
				kvString("task.dependency.name", ref.Name),
			},
		})
	}
	return links
}

func kvString(key string, value string) *commonv1.KeyValue {
	return &commonv1.KeyValue{
		Key: key,
		Value: &commonv1.AnyValue{
			Value: &commonv1.AnyValue_StringValue{StringValue: value},
		},
	}
}

func kvStrings(key string, values []string) *commonv1.KeyValue {
	attrValues := make([]*commonv1.AnyValue, 0, len(values))
	for _, value := range values {
		attrValues = append(attrValues, &commonv1.AnyValue{
			Value: &commonv1.AnyValue_StringValue{StringValue: value},
		})
	}
	return &commonv1.KeyValue{
		Key: key,
		Value: &commonv1.AnyValue{
			Value: &commonv1.AnyValue_ArrayValue{
				ArrayValue: &commonv1.ArrayValue{Values: attrValues},
			},
		},
	}
}
