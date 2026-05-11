package main

import (
	"context"
	"fmt"

	"dagger/tempo-pipeline/internal/dagger"
)

type TempoPipeline struct{}

const (
	ghostTraceID = "0af7651916cd43dd8448eb211c80319c"
	ghostSpanID  = "b9c7c989f97918e1"
)

// RunTempo returns a service that runs Grafana Tempo with a local config.
func (m *TempoPipeline) RunTempo(config *dagger.File) *dagger.Service {
	return dag.Container().
		From("grafana/tempo:latest").
		WithFile("/etc/tempo.yaml", config).
		WithExposedPort(3200). // HTTP
		WithExposedPort(4317). // OTLP gRPC
		WithExposedPort(4318). // OTLP HTTP
		WithExec([]string{"/tempo", "-config.file=/etc/tempo.yaml"}).
		AsService()
}

func (m *TempoPipeline) tempo() *dagger.Service {
	return m.RunTempo(dag.CurrentModule().Source().File("tempo-config.yaml"))
}

// GhostTrace sends a deterministic synthetic OTLP trace to Tempo.
func (m *TempoPipeline) GhostTrace(ctx context.Context) (string, error) {
	_, err := dag.Container().
		From("curlimages/curl:8.9.1").
		WithServiceBinding("tempo", m.tempo()).
		WithExec([]string{"sh", "-c", fmt.Sprintf(`
START=$(( $(date +%%s) * 1000000000 ))
END=$(( START + 1000000000 ))
curl -sf -X POST http://tempo:4318/v1/traces \
  -H 'Content-Type: application/json' \
  -d "{\"resourceSpans\":[{\"resource\":{\"attributes\":[{\"key\":\"service.name\",\"value\":{\"stringValue\":\"ghost-trace\"}}]},\"scopeSpans\":[{\"spans\":[{\"traceId\":\"%s\",\"spanId\":\"%s\",\"name\":\"ghost-span\",\"kind\":1,\"startTimeUnixNano\":\"$START\",\"endTimeUnixNano\":\"$END\",\"status\":{}}]}]}]}"
`, ghostTraceID, ghostSpanID)}).
		Sync(ctx)
	return ghostTraceID, err
}

// Check asserts the ghost trace is present in Tempo.
//
// +check
func (m *TempoPipeline) Check(ctx context.Context) (string, error) {
	return dag.Container().
		From("curlimages/curl:8.9.1").
		WithServiceBinding("tempo", m.tempo()).
		WithExec([]string{"sh", "-c", fmt.Sprintf(`
result=$(curl -sf "http://tempo:3200/api/traces/%s")
echo "$result" | grep -q 'ghost-span' \
  && echo "ok: ghost trace %s confirmed in Tempo" \
  && exit 0
echo "not found — result: $result" >&2
exit 1`, ghostTraceID, ghostTraceID)}).
		Stdout(ctx)
}
