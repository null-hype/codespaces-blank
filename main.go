package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"dagger/tempo-pipeline/domain"
	"dagger/tempo-pipeline/internal/compiler"
	"dagger/tempo-pipeline/internal/dagger"
	"dagger/tempo-pipeline/internal/evidence"
	"dagger/tempo-pipeline/internal/otel"
	"dagger/tempo-pipeline/internal/planner"
	"dagger/tempo-pipeline/internal/taskwarrior"
	"dagger/tempo-pipeline/projections"
)

type TempoPipeline struct{}

// RunTempo returns a service that runs Grafana Tempo and emits evidence for the run-tempo task.
func (m *TempoPipeline) RunTempo(
	ctx context.Context,
	config *dagger.File,
) (*dagger.Service, error) {
	svc := m.runTempoService(config)
	if err := EmitEvidence(ctx, svc, domain.RunTempoTask, projections.RunTempo); err != nil {
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

// Tempo returns a service that runs Grafana Tempo with this module's default config and emits evidence for the tempo task.
func (m *TempoPipeline) Tempo(ctx context.Context) (*dagger.Service, error) {
	svc, err := m.RunTempo(ctx, dag.CurrentModule().Source().File("tempo-config.yaml"))
	if err != nil {
		return nil, err
	}
	if err := EmitEvidence(ctx, svc, domain.TempoTask, projections.Tempo); err != nil {
		return nil, err
	}
	return svc, nil
}

// ValidatePlan validates a Taskwarrior JSONL export against the Pkl-modeled work plan.
func (m *TempoPipeline) ValidatePlan(
	ctx context.Context,
	// Taskwarrior JSONL from `task export` with json.array=off.
	//
	// +defaultPath="task-dag.jsonl"
	taskExport *dagger.File,
) (string, error) {
	result, err := validateTaskExport(ctx, taskExport)
	if err != nil {
		return "", err
	}
	return planner.Summary(result), nil
}

// GenerateDag compiles Taskwarrior JSONL into a generated DAG directory.
//
// It is a pure compiler: it does not execute Dagger tasks, start Tempo, emit OTEL, or verify evidence.
func (m *TempoPipeline) GenerateDag(
	ctx context.Context,
	// Taskwarrior JSONL from `task export` with json.array=off.
	//
	// +defaultPath="task-dag.jsonl"
	taskExport *dagger.File,
	// Domain Pkl config. Defaults to the bundled pkl/plans/tempo-pipeline.pkl model.
	//
	// +optional
	domainConfig *dagger.File,
) (*dagger.Directory, error) {
	contents, err := defaultTaskExport(taskExport).Contents(ctx)
	if err != nil {
		return nil, fmt.Errorf("read task export: %w", err)
	}
	tasks, err := taskwarrior.ParseExport(contents)
	if err != nil {
		return nil, err
	}
	cfg, err := loadDomainConfig(ctx, domainConfig)
	if err != nil {
		return nil, err
	}
	result := compiler.Generate(tasks, cfg)
	artifacts, err := compiler.RenderArtifacts(result)
	if err != nil {
		return nil, err
	}
	return dag.Directory().
		WithNewFile("dag.json", artifacts.DAGJSON).
		WithNewFile("diagnostics.json", artifacts.DiagnosticsJSON).
		WithNewFile("normalized-taskwarrior.jsonl", artifacts.NormalizedTaskwarriorJSONL).
		WithNewFile("evidence-contract.json", artifacts.EvidenceContractJSON).
		WithNewFile("otel-projection.json", artifacts.OtelProjectionJSON).
		WithNewFile("runbook.md", artifacts.RunbookMarkdown), nil
}

// Check validates the plan, runs the Dagger functions, and verifies required evidence.
//
// +check
func (m *TempoPipeline) Check(
	ctx context.Context,
	// Taskwarrior JSONL from `task export` with json.array=off.
	//
	// +defaultPath="task-dag.jsonl"
	taskExport *dagger.File,
) (string, error) {
	evidence.Reset()
	if _, err := validateTaskExport(ctx, taskExport); err != nil {
		return "", err
	}

	svc, err := m.Tempo(ctx)
	if err != nil {
		return "", err
	}
	if err := evidence.Verify(projections.ExpectedEvidence); err != nil {
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

echo "ok: validated %s and confirmed required OTEL spans in Tempo"`, projections.TraceID, projections.TraceID, projections.TraceID, domain.TempoPipelinePlan.Id)}).
		Stdout(ctx)
}

func validateTaskExport(ctx context.Context, taskExport *dagger.File) (compiler.Result, error) {
	contents, err := defaultTaskExport(taskExport).Contents(ctx)
	if err != nil {
		return compiler.Result{}, fmt.Errorf("read task export: %w", err)
	}
	result, err := planner.ValidateCurrentPlanExport(contents)
	if err != nil && compiler.HasErrors(result) {
		return result, fmt.Errorf("%w: see generate-dag diagnostics for details", err)
	}
	return result, err
}

func defaultTaskExport(taskExport *dagger.File) *dagger.File {
	if taskExport != nil {
		return taskExport
	}
	return dag.CurrentModule().Source().File("task-dag.jsonl")
}

func expectedSpanNames() []string {
	names := make([]string, 0, len(projections.ExpectedOtel))
	for _, span := range projections.ExpectedOtel {
		names = append(names, span.Name)
	}
	return names
}

func loadDomainConfig(ctx context.Context, domainConfig *dagger.File) (domain.WorkDomain, error) {
	if domainConfig == nil {
		return domain.Current, nil
	}
	out, err := dag.Container().
		From("buildpack-deps:bookworm-curl").
		WithDirectory("/work/pkl", dag.CurrentModule().Source().Directory("pkl")).
		WithFile("/work/pkl/plans/domain-config.pkl", domainConfig).
		WithExec([]string{"sh", "-c", `
set -eu
curl --fail-with-body --show-error --silent --location \
  https://github.com/apple/pkl/releases/download/0.31.1/pkl-linux-amd64 \
  --output /tmp/pkl
chmod +x /tmp/pkl
/tmp/pkl eval /work/pkl/plans/domain-config.pkl --format json > /tmp/domain.json
`}).
		File("/tmp/domain.json").
		Contents(ctx)
	if err != nil {
		return domain.WorkDomain{}, fmt.Errorf("evaluate domain config: %w", err)
	}
	var cfg domain.WorkDomain
	if err := json.Unmarshal([]byte(out), &cfg); err != nil {
		return domain.WorkDomain{}, fmt.Errorf("decode domain config: %w", err)
	}
	return cfg, nil
}

// EmitEvidence records JSONL evidence and emits the OTEL projection for one work-plan task.
func EmitEvidence(ctx context.Context, svc *dagger.Service, task domain.Task, span projections.OtelSpan) error {
	evidence.RecordTask(domain.TempoPipelinePlan.Id, task, "jsonl-event")

	tracePayload, err := otel.PayloadBase64(span)
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

echo "waiting for Tempo readiness before emitting evidence for %s"
i=0
until curl_tempo http://tempo:3200/ready >/dev/null; do
  i=$((i + 1))
  if [ "$i" -ge 10 ]; then
    echo "Tempo did not become ready after $i attempts" >&2
    exit 1
  fi
  sleep 1
done

echo "emitting OTEL evidence for %s"
base64 -d /tmp/task-trace.pb.b64 > /tmp/task-trace.pb
curl_tempo -X POST http://tempo:4318/v1/traces \
  -H 'Content-Type: application/x-protobuf' \
  --data-binary @/tmp/task-trace.pb >/dev/null
`, task.Id, task.Id)}).
		Sync(ctx)
	return err
}
