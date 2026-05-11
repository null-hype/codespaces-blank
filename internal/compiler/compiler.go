package compiler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"dagger/tempo-pipeline/domain"
	"dagger/tempo-pipeline/internal/taskwarrior"
)

const defaultPlanID = "plan.generated"

type ArtifactSet struct {
	DAGJSON                    string
	DiagnosticsJSON            string
	NormalizedTaskwarriorJSONL string
	EvidenceContractJSON       string
	OtelProjectionJSON         string
	RunbookMarkdown            string
}

type Diagnostic struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	TaskID   string `json:"taskID,omitempty"`
	UUID     string `json:"uuid,omitempty"`
	Edge     *Edge  `json:"edge,omitempty"`
}

type Node struct {
	ID          string                       `json:"id"`
	UUID        string                       `json:"uuid"`
	Description string                       `json:"description"`
	Project     string                       `json:"project,omitempty"`
	Status      string                       `json:"status"`
	Tags        []string                     `json:"tags,omitempty"`
	Bound       bool                         `json:"bound"`
	BindingRule string                       `json:"bindingRule,omitempty"`
	Actor       string                       `json:"actor,omitempty"`
	Capability  string                       `json:"capability,omitempty"`
	Command     *domain.Command              `json:"command,omitempty"`
	Evidence    []domain.EvidenceRequirement `json:"evidence,omitempty"`
}

type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
	UUID string `json:"uuid,omitempty"`
}

type Result struct {
	PlanID           string             `json:"planID"`
	Project          string             `json:"project,omitempty"`
	Scope            string             `json:"scope,omitempty"`
	WorkPlan         domain.WorkPlan    `json:"workPlan"`
	Nodes            []Node             `json:"nodes"`
	Edges            []Edge             `json:"edges"`
	TopologicalOrder []string           `json:"topologicalOrder"`
	Diagnostics      []Diagnostic       `json:"diagnostics"`
	Normalized       []NormalizedTask   `json:"-"`
	Evidence         []EvidenceContract `json:"-"`
	OtelProjection   OtelProjection     `json:"-"`
}

type NormalizedTask struct {
	UUID        string   `json:"uuid"`
	Description string   `json:"description"`
	Project     string   `json:"project,omitempty"`
	Status      string   `json:"status"`
	Tags        []string `json:"tags,omitempty"`
	Depends     []string `json:"depends,omitempty"`
}

type EvidenceContract struct {
	TaskID          string          `json:"taskID"`
	TaskwarriorUUID string          `json:"taskwarriorUuid"`
	Kind            string          `json:"kind"`
	Required        bool            `json:"required"`
	Producer        *domain.Command `json:"producer,omitempty"`
}

type OtelProjection struct {
	PlanID  string     `json:"planID"`
	TraceID string     `json:"traceID"`
	Spans   []OtelSpan `json:"spans"`
}

type OtelSpan struct {
	TaskID          string    `json:"taskID"`
	GoName          string    `json:"goName,omitempty"`
	Name            string    `json:"name"`
	TaskwarriorUUID string    `json:"taskwarriorUuid"`
	TraceID         string    `json:"traceID"`
	SpanID          string    `json:"spanID"`
	Required        bool      `json:"required"`
	Project         string    `json:"project,omitempty"`
	Status          string    `json:"status"`
	Tags            []string  `json:"tags,omitempty"`
	Depends         []SpanRef `json:"depends,omitempty"`
}

type SpanRef struct {
	TaskID string `json:"taskID"`
	Name   string `json:"name"`
	SpanID string `json:"spanID"`
}

func Generate(tasks []taskwarrior.Task, cfg domain.WorkDomain) Result {
	base := basePlan(cfg)
	result := Result{
		PlanID:      base.Id,
		Project:     base.Project,
		Scope:       base.Scope,
		Nodes:       []Node{},
		Edges:       []Edge{},
		Diagnostics: []Diagnostic{},
		WorkPlan: domain.WorkPlan{
			Id:      base.Id,
			Project: base.Project,
			Scope:   base.Scope,
			Tasks:   []domain.Task{},
		},
	}
	if result.PlanID == "" {
		result.PlanID = defaultPlanID
		result.WorkPlan.Id = result.PlanID
	}

	result.Normalized = normalizeTasks(tasks)
	uuidToNode := map[string]*Node{}
	idCounts := map[string]int{}
	actorCaps := actorCapabilities(cfg)
	capabilities := capabilitySet(cfg)

	for _, task := range tasks {
		status := task.Status
		if status == "" {
			status = "pending"
		}
		node := Node{
			UUID:        task.UUID,
			Description: task.Description,
			Project:     task.Project,
			Status:      status,
			Tags:        sortedStrings(task.Tags),
		}
		if task.UUID == "" {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Severity: "error",
				Code:     "MISSING_UUID",
				Message:  fmt.Sprintf("task %q has no Taskwarrior uuid", task.Description),
			})
		}
		if _, exists := uuidToNode[task.UUID]; task.UUID != "" && exists {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Severity: "error",
				Code:     "DUPLICATE_UUID",
				Message:  fmt.Sprintf("task export contains duplicate uuid %s", task.UUID),
				UUID:     task.UUID,
			})
		}

		node.ID = uniqueTaskID(slugTaskID(task.Description, task.UUID), idCounts)
		bindByConvention(&node, task)
		validateBinding(&result, node, actorCaps, capabilities)

		uuidToNode[task.UUID] = &node
		result.Nodes = append(result.Nodes, node)
		result.WorkPlan.Tasks = append(result.WorkPlan.Tasks, domain.Task{
			Id:              node.ID,
			TaskwarriorUuid: task.UUID,
			Description:     task.Description,
			Project:         result.Project,
			Status:          status,
			Tags:            append([]string(nil), node.Tags...),
			Actor:           node.Actor,
			Capability:      node.Capability,
			Command:         cloneCommand(node.Command),
			Evidence:        append([]domain.EvidenceRequirement(nil), node.Evidence...),
		})
	}

	result.Edges = buildEdges(&result, tasks, uuidToNode)
	applyDepends(&result)
	result.TopologicalOrder = topologicalOrder(&result)
	result.Evidence = buildEvidence(result.Nodes)
	result.OtelProjection = buildOtelProjection(result.PlanID, result.Nodes, result.Edges)
	return result
}

func RenderArtifacts(result Result) (ArtifactSet, error) {
	diagnostics := struct {
		OK          bool         `json:"ok"`
		ErrorCount  int          `json:"errorCount"`
		WarnCount   int          `json:"warnCount"`
		Diagnostics []Diagnostic `json:"diagnostics"`
	}{
		OK:          !HasErrors(result),
		Diagnostics: result.Diagnostics,
	}
	for _, d := range result.Diagnostics {
		switch d.Severity {
		case "error":
			diagnostics.ErrorCount++
		case "warning":
			diagnostics.WarnCount++
		}
	}

	dagJSON, err := marshalIndent(result)
	if err != nil {
		return ArtifactSet{}, err
	}
	diagnosticsJSON, err := marshalIndent(diagnostics)
	if err != nil {
		return ArtifactSet{}, err
	}
	evidenceJSON, err := marshalIndent(struct {
		PlanID   string             `json:"planID"`
		Evidence []EvidenceContract `json:"evidence"`
	}{PlanID: result.PlanID, Evidence: result.Evidence})
	if err != nil {
		return ArtifactSet{}, err
	}
	otelJSON, err := marshalIndent(result.OtelProjection)
	if err != nil {
		return ArtifactSet{}, err
	}

	return ArtifactSet{
		DAGJSON:                    dagJSON,
		DiagnosticsJSON:            diagnosticsJSON,
		NormalizedTaskwarriorJSONL: normalizedJSONL(result.Normalized),
		EvidenceContractJSON:       evidenceJSON,
		OtelProjectionJSON:         otelJSON,
		RunbookMarkdown:            runbook(result, diagnostics.ErrorCount, diagnostics.WarnCount),
	}, nil
}

func HasErrors(result Result) bool {
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Severity == "error" {
			return true
		}
	}
	return false
}

func Summary(result Result) string {
	status := "ok"
	if HasErrors(result) {
		status = "diagnostics"
	}
	return fmt.Sprintf("%s: compiled %s with %d tasks, %d edges, %d diagnostics", status, result.PlanID, len(result.Nodes), len(result.Edges), len(result.Diagnostics))
}

func basePlan(cfg domain.WorkDomain) domain.WorkPlan {
	if len(cfg.Plans) > 0 {
		return cfg.Plans[0]
	}
	return domain.WorkPlan{Id: defaultPlanID}
}

func normalizeTasks(tasks []taskwarrior.Task) []NormalizedTask {
	normalized := make([]NormalizedTask, 0, len(tasks))
	for _, task := range tasks {
		status := task.Status
		if status == "" {
			status = "pending"
		}
		normalized = append(normalized, NormalizedTask{
			UUID:        task.UUID,
			Description: task.Description,
			Project:     task.Project,
			Status:      status,
			Tags:        sortedStrings(task.Tags),
			Depends:     sortedStrings(task.Depends),
		})
	}
	return normalized
}

func slugTaskID(description, uuid string) string {
	slug := strings.Trim(strings.Map(func(r rune) rune {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			return unicode.ToLower(r)
		case r == '-' || r == '_' || unicode.IsSpace(r):
			return '-'
		default:
			return -1
		}
	}, description), "-")
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	if slug == "" {
		slug = "task"
	}
	if uuid != "" && slug == "task" {
		slug += "-" + shortUUID(uuid)
	}
	return "task." + slug
}

func bindByConvention(node *Node, task taskwarrior.Task) {
	if contains(task.Tags, "dagger") {
		node.Actor = "actor.dagger"
	}
	switch task.Description {
	case "run-tempo":
		node.Command = &domain.Command{Kind: "dagger-function", Name: "RunTempo"}
		node.Capability = "cap.dagger-service"
		node.Evidence = []domain.EvidenceRequirement{
			{Id: "ev.run-tempo.jsonl", Kind: "jsonl-event", Required: true},
			{Id: "ev.run-tempo.otel", Kind: "otel-span", Required: true},
		}
	case "tempo":
		node.Command = &domain.Command{Kind: "dagger-function", Name: "Tempo"}
		node.Capability = "cap.dagger-service"
		node.Evidence = []domain.EvidenceRequirement{
			{Id: "ev.tempo.jsonl", Kind: "jsonl-event", Required: true},
			{Id: "ev.tempo.otel", Kind: "otel-span", Required: true},
		}
	case "check":
		node.Command = &domain.Command{Kind: "dagger-function", Name: "Check"}
		node.Capability = "cap.evidence-check"
		node.Evidence = []domain.EvidenceRequirement{
			{Id: "ev.check.diagnostic", Kind: "diagnostic", Required: true},
		}
	}
	node.Bound = node.Actor != "" && node.Command != nil && node.Capability != "" && len(node.Evidence) > 0
}

func uniqueTaskID(id string, counts map[string]int) string {
	if counts[id] == 0 {
		counts[id] = 1
		return id
	}
	counts[id]++
	return fmt.Sprintf("%s-%d", id, counts[id])
}

func buildEdges(result *Result, tasks []taskwarrior.Task, uuidToNode map[string]*Node) []Edge {
	var edges []Edge
	for _, task := range tasks {
		to := uuidToNode[task.UUID]
		if to == nil {
			continue
		}
		for _, depUUID := range task.Depends {
			if depUUID == task.UUID {
				result.Diagnostics = append(result.Diagnostics, Diagnostic{
					Severity: "error",
					Code:     "SELF_DEPENDENCY",
					Message:  fmt.Sprintf("task %s depends on itself", to.ID),
					TaskID:   to.ID,
					UUID:     task.UUID,
				})
				continue
			}
			from := uuidToNode[depUUID]
			if from == nil {
				edge := Edge{To: to.ID, UUID: depUUID}
				result.Diagnostics = append(result.Diagnostics, Diagnostic{
					Severity: "error",
					Code:     "UNKNOWN_DEPENDENCY",
					Message:  fmt.Sprintf("task %s depends on unknown uuid %s", to.ID, depUUID),
					TaskID:   to.ID,
					UUID:     task.UUID,
					Edge:     &edge,
				})
				continue
			}
			edges = append(edges, Edge{From: from.ID, To: to.ID, UUID: depUUID})
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From == edges[j].From {
			return edges[i].To < edges[j].To
		}
		return edges[i].From < edges[j].From
	})
	return edges
}

func applyDepends(result *Result) {
	deps := map[string][]string{}
	for _, edge := range result.Edges {
		deps[edge.To] = append(deps[edge.To], edge.From)
	}
	for i := range result.WorkPlan.Tasks {
		task := &result.WorkPlan.Tasks[i]
		task.DependsOn = sortedStrings(deps[task.Id])
	}
}

func topologicalOrder(result *Result) []string {
	indegree := map[string]int{}
	outgoing := map[string][]string{}
	for _, node := range result.Nodes {
		indegree[node.ID] = 0
	}
	for _, edge := range result.Edges {
		indegree[edge.To]++
		outgoing[edge.From] = append(outgoing[edge.From], edge.To)
	}
	var ready []string
	for id, degree := range indegree {
		if degree == 0 {
			ready = append(ready, id)
		}
	}
	sort.Strings(ready)
	var order []string
	for len(ready) > 0 {
		id := ready[0]
		ready = ready[1:]
		order = append(order, id)
		sort.Strings(outgoing[id])
		for _, next := range outgoing[id] {
			indegree[next]--
			if indegree[next] == 0 {
				ready = append(ready, next)
				sort.Strings(ready)
			}
		}
	}
	if len(order) != len(result.Nodes) {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Severity: "error",
			Code:     "CYCLE",
			Message:  "task dependency graph contains a cycle",
		})
	}
	return order
}

func validateBinding(result *Result, node Node, actorCaps map[string]map[string]bool, capabilities map[string]bool) {
	if node.Command == nil {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Severity: "error",
			Code:     "UNBOUND_COMMAND",
			Message:  fmt.Sprintf("task %s has no command bound by convention", node.ID),
			TaskID:   node.ID,
			UUID:     node.UUID,
		})
	}
	if node.Capability == "" || !capabilities[node.Capability] {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Severity: "error",
			Code:     "UNBOUND_CAPABILITY",
			Message:  fmt.Sprintf("task %s has no valid capability bound by convention", node.ID),
			TaskID:   node.ID,
			UUID:     node.UUID,
		})
	}
	if node.Actor != "" && node.Capability != "" {
		caps, ok := actorCaps[node.Actor]
		if !ok || !caps[node.Capability] {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Severity: "error",
				Code:     "ACTOR_NOT_ALLOWED_CAPABILITY",
				Message:  fmt.Sprintf("actor %s is not allowed to use capability %s for task %s", node.Actor, node.Capability, node.ID),
				TaskID:   node.ID,
				UUID:     node.UUID,
			})
		}
	}
	if len(node.Evidence) == 0 {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Severity: "error",
			Code:     "MISSING_EVIDENCE_REQUIREMENT",
			Message:  fmt.Sprintf("task %s has no evidence requirement bound by convention", node.ID),
			TaskID:   node.ID,
			UUID:     node.UUID,
		})
	}
}

func actorCapabilities(cfg domain.WorkDomain) map[string]map[string]bool {
	result := map[string]map[string]bool{}
	for _, actor := range cfg.Actors {
		result[actor.Id] = map[string]bool{}
		for _, capID := range actor.AllowedCapabilities {
			result[actor.Id][capID] = true
		}
	}
	return result
}

func capabilitySet(cfg domain.WorkDomain) map[string]bool {
	result := map[string]bool{}
	for _, cap := range cfg.Capabilities {
		result[cap.Id] = true
	}
	return result
}

func buildEvidence(nodes []Node) []EvidenceContract {
	var evidence []EvidenceContract
	for _, node := range nodes {
		for _, req := range node.Evidence {
			evidence = append(evidence, EvidenceContract{
				TaskID:          node.ID,
				TaskwarriorUUID: node.UUID,
				Kind:            req.Kind,
				Required:        req.Required,
				Producer:        cloneCommand(node.Command),
			})
		}
	}
	sort.Slice(evidence, func(i, j int) bool {
		if evidence[i].TaskID == evidence[j].TaskID {
			return evidence[i].Kind < evidence[j].Kind
		}
		return evidence[i].TaskID < evidence[j].TaskID
	})
	return evidence
}

func buildOtelProjection(planID string, nodes []Node, edges []Edge) OtelProjection {
	traceID := traceID(nodes)
	spanByTask := map[string]OtelSpan{}
	var spans []OtelSpan
	for _, node := range nodes {
		req, ok := otelEvidence(node)
		if !ok {
			continue
		}
		span := OtelSpan{
			TaskID:          node.ID,
			Name:            node.Description,
			TaskwarriorUUID: node.UUID,
			TraceID:         traceID,
			SpanID:          digestHex("span:"+node.UUID, 8),
			Required:        req.Required,
			Project:         node.Project,
			Status:          node.Status,
			Tags:            append([]string(nil), node.Tags...),
		}
		if node.Command != nil {
			span.GoName = node.Command.Name
		}
		spanByTask[node.ID] = span
		spans = append(spans, span)
	}
	for i := range spans {
		for _, edge := range edges {
			if edge.To != spans[i].TaskID {
				continue
			}
			parent, ok := spanByTask[edge.From]
			if !ok {
				continue
			}
			spans[i].Depends = append(spans[i].Depends, SpanRef{TaskID: parent.TaskID, Name: parent.Name, SpanID: parent.SpanID})
		}
		sort.Slice(spans[i].Depends, func(a, b int) bool {
			return spans[i].Depends[a].TaskID < spans[i].Depends[b].TaskID
		})
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i].TaskID < spans[j].TaskID })
	if spans == nil {
		spans = []OtelSpan{}
	}
	return OtelProjection{PlanID: planID, TraceID: traceID, Spans: spans}
}

func otelEvidence(node Node) (domain.EvidenceRequirement, bool) {
	for _, req := range node.Evidence {
		if req.Kind == "otel-span" {
			return req, true
		}
	}
	return domain.EvidenceRequirement{}, false
}

func traceID(nodes []Node) string {
	uuids := make([]string, 0, len(nodes))
	for _, node := range nodes {
		uuids = append(uuids, node.UUID)
	}
	sort.Strings(uuids)
	return digestHex("trace:"+strings.Join(uuids, "\n"), 16)
}

func digestHex(input string, bytes int) string {
	sum := sha256.Sum256([]byte(input))
	return hex.EncodeToString(sum[:bytes])
}

func normalizedJSONL(tasks []NormalizedTask) string {
	var lines []string
	for _, task := range tasks {
		b, _ := json.Marshal(task)
		lines = append(lines, string(b))
	}
	return strings.Join(lines, "\n") + "\n"
}

func runbook(result Result, errors, warnings int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", result.PlanID)
	fmt.Fprintf(&b, "- tasks: %d\n- edges: %d\n- diagnostics: %d error(s), %d warning(s)\n\n", len(result.Nodes), len(result.Edges), errors, warnings)
	b.WriteString("## Topological Order\n\n")
	for _, id := range result.TopologicalOrder {
		fmt.Fprintf(&b, "- %s\n", id)
	}
	if len(result.TopologicalOrder) == 0 {
		b.WriteString("- no complete topological order\n")
	}
	b.WriteString("\n## Evidence Contract\n\n")
	for _, ev := range result.Evidence {
		fmt.Fprintf(&b, "- %s requires %s", ev.TaskID, ev.Kind)
		if ev.Producer != nil {
			fmt.Fprintf(&b, " from %s", ev.Producer.Name)
		}
		b.WriteString("\n")
	}
	b.WriteString("\n## Diagnostics\n\n")
	if len(result.Diagnostics) == 0 {
		b.WriteString("- none\n")
	} else {
		for _, d := range result.Diagnostics {
			fmt.Fprintf(&b, "- [%s] %s: %s\n", d.Severity, d.Code, d.Message)
		}
	}
	return b.String()
}

func marshalIndent(v any) (string, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b) + "\n", nil
}

func cloneCommand(cmd *domain.Command) *domain.Command {
	if cmd == nil {
		return nil
	}
	c := *cmd
	return &c
}

func sortedStrings(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func shortUUID(uuid string) string {
	if len(uuid) <= 8 {
		return uuid
	}
	return uuid[:8]
}
