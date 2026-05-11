package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"dagger/tempo-pipeline/spans"

	"dagger/tempo-pipeline/internal/dagger"
	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	resourcev1 "go.opentelemetry.io/proto/otlp/resource/v1"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"
)

type TempoPipeline struct{}

const taskTraceNamespace = "tempo-pipeline/task-dag"

// RunTempo returns a service that runs Grafana Tempo and emits its typed task span.
func (m *TempoPipeline) RunTempo(
	ctx context.Context,
	config *dagger.File,
) (*dagger.Service, error) {
	svc := m.runTempoService(config)
	if err := EmitOtel(ctx, svc, spans.RunTempo); err != nil {
		return nil, err
	}
	return svc, nil
}

func (m *TempoPipeline) runTempoService(config *dagger.File) *dagger.Service {
	return dag.Container().
		From("grafana/tempo:latest").
		WithFile("/etc/tempo.yaml", config).
		WithExposedPort(3200, dagger.ContainerWithExposedPortOpts{ExperimentalSkipHealthcheck: true}). // HTTP
		WithExposedPort(4317, dagger.ContainerWithExposedPortOpts{ExperimentalSkipHealthcheck: true}). // OTLP gRPC
		WithExposedPort(4318, dagger.ContainerWithExposedPortOpts{ExperimentalSkipHealthcheck: true}). // OTLP HTTP
		AsService(dagger.ContainerAsServiceOpts{Args: []string{"/tempo", "-config.file=/etc/tempo.yaml"}})
}

// Tempo returns a service that runs Grafana Tempo with this module's default config and emits its typed task span.
func (m *TempoPipeline) Tempo(ctx context.Context) (*dagger.Service, error) {
	svc, err := m.RunTempo(ctx, dag.CurrentModule().Source().File("tempo-config.yaml"))
	if err != nil {
		return nil, err
	}
	if err := EmitOtel(ctx, svc, spans.Tempo); err != nil {
		return nil, err
	}
	return svc, nil
}

// Check asserts that required typed task spans emitted by Dagger functions are present in Tempo.
//
// +check
func (m *TempoPipeline) Check(ctx context.Context) (string, error) {
	svc, err := m.Tempo(ctx)
	if err != nil {
		return "", err
	}

	return dag.Container().
		From("curlimages/curl:8.9.1").
		WithServiceBinding("tempo", svc).
		WithNewFile("/tmp/expected-spans.txt", strings.Join(expectedSpanNames(), "\n")+"\n").
		WithExec([]string{"sh", "-c", fmt.Sprintf(`
set -eu

curl_tempo() {
  curl --fail-with-body --show-error --silent --connect-timeout 2 --max-time 5 "$@"
}

all_spans_present() {
  while IFS= read -r span; do
    [ -z "$span" ] && continue
    echo "$result" | grep -F -q -- "\"name\":\"$span\"" || return 1
  done < /tmp/expected-spans.txt
}

echo "waiting for Tempo readiness"
i=0
until curl_tempo http://tempo:3200/ready >/dev/null; do
  i=$((i + 1))
  if [ "$i" -ge 10 ]; then
    echo "Tempo did not become ready after $i attempts" >&2
    exit 1
  fi
  sleep 1
done

echo "querying task DAG trace %s"
last_result=""
i=0
until result=$(curl_tempo "http://tempo:3200/api/traces/%s") && all_spans_present; do
  last_result="${result:-}"
  i=$((i + 1))
  if [ "$i" -ge 10 ]; then
    echo "task DAG trace %s did not contain every expected span after $i attempts" >&2
    echo "last result: $last_result" >&2
    exit 1
  fi
  sleep 1
done

echo "ok: task DAG trace %s confirmed in Tempo"`, spans.TraceID, spans.TraceID, spans.TraceID, spans.TraceID)}).
		Stdout(ctx)
}

func expectedSpanNames() []string {
	names := make([]string, 0, len(spans.Expected))
	for _, span := range spans.Expected {
		names = append(names, span.Name)
	}
	return names
}

// EmitOtel emits one typed task span contract to the bound Tempo service.
func EmitOtel(ctx context.Context, svc *dagger.Service, span spans.TaskSpan) error {
	tracePayload, err := taskSpanPayloadBase64(span)
	if err != nil {
		return err
	}

	_, err = dag.Container().
		From("curlimages/curl:8.9.1").
		WithServiceBinding("tempo", svc).
		WithNewFile("/tmp/task-trace.pb.b64", tracePayload).
		WithExec([]string{"sh", "-c", fmt.Sprintf(`
set -eu

curl_tempo() {
  curl --fail-with-body --show-error --silent --connect-timeout 2 --max-time 5 "$@"
}

echo "waiting for Tempo readiness before emitting task span %s"
i=0
until curl_tempo http://tempo:3200/ready >/dev/null; do
  i=$((i + 1))
  if [ "$i" -ge 10 ]; then
    echo "Tempo did not become ready after $i attempts" >&2
    exit 1
  fi
  sleep 1
done

echo "emitting task span %s"
base64 -d /tmp/task-trace.pb.b64 > /tmp/task-trace.pb
curl_tempo -X POST http://tempo:4318/v1/traces \
  -H 'Content-Type: application/x-protobuf' \
  --data-binary @/tmp/task-trace.pb >/dev/null
`, span.Name, span.Name)}).
		Sync(ctx)
	return err
}

func taskSpanPayloadBase64(span spans.TaskSpan) (string, error) {
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
							Version: "pkl-task-span-contract",
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

func spanAttributes(span spans.TaskSpan) []*commonv1.KeyValue {
	attrs := []*commonv1.KeyValue{
		kvString("task.uuid", span.Uuid),
		kvString("task.description", span.Name),
		kvString("task.go_name", span.GoName),
		kvString("task.emitter", "dagger-function"),
		kvString("task.trace.namespace", taskTraceNamespace),
	}
	if span.Status != nil {
		attrs = append(attrs, kvString("task.status", *span.Status))
	}
	if span.Project != nil {
		attrs = append(attrs, kvString("task.project", *span.Project))
	}
	if len(span.Tags) > 0 {
		attrs = append(attrs, kvStrings("task.tags", span.Tags))
	}
	if len(span.Depends) > 0 {
		depUUIDs := make([]string, 0, len(span.Depends))
		for _, dep := range span.Depends {
			depUUIDs = append(depUUIDs, dep.Uuid)
		}
		attrs = append(attrs, kvStrings("task.depends", depUUIDs))
	}
	return attrs
}

func spanLinks(traceID string, refs []spans.SpanRef) []*tracev1.Span_Link {
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
				kvString("task.dependency.name", ref.Name),
				kvString("task.dependency.uuid", ref.Uuid),
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
