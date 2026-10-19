package service

import (
	"fmt"
	"time"

	"github.com/rafapasa/etoolscode/internal/config"
	"github.com/rafapasa/etoolscode/internal/core"
	"github.com/rafapasa/etoolscode/internal/core/tools"
	"github.com/rafapasa/etoolscode/internal/dto"
	"github.com/rafapasa/etoolscode/internal/infra"
)

type EventType string

const (
	EventPlanGenerated    EventType = "plan_generated"
	EventStepStarted      EventType = "step_started"
	EventStepCompleted    EventType = "step_completed"
	EventDiffReady        EventType = "diff_ready"
	EventApprovalRequired EventType = "approval_required"
	EventCompleted        EventType = "completed"
	EventFailed           EventType = "failed"
	EventLog              EventType = "log"
)

type Event struct {
	Type      EventType `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	PlanID    string    `json:"plan_id,omitempty"`
	StepID    string    `json:"step_id,omitempty"`
	Message   string    `json:"message"`
	Data      any       `json:"data,omitempty"`
}

type OrchestratorV2 struct {
	cfg          *config.Config
	llm          core.LlmInterface
	planner      *PlannerService
	rag          *RAGService
	tokenCounter *TokenCounter
	audit        *AuditService
	logger       *infra.Logger
	onEvent      func(Event)
}

func NewOrchestratorV2(cfg *config.Config, llm core.LlmInterface, onEvent func(Event)) *OrchestratorV2 {
	if cfg == nil {
		cfg = config.NewConfig()
	}
	return &OrchestratorV2{
		cfg: cfg, llm: llm,
		planner:      NewPlannerService(NewRAGService("")),
		rag:          NewRAGService(""),
		tokenCounter: NewTokenCounter(),
		audit:        NewAuditService(),
		logger:       infra.L(),
		onEvent:      onEvent,
	}
}

func (o *OrchestratorV2) emit(ev Event) {
	ev.Timestamp = time.Now()
	if o.onEvent != nil {
		o.onEvent(ev)
	}
}

func (o *OrchestratorV2) StartWithPlan(projectRoot, issueID string, issue dto.Request, dryRun bool) (*Plan, error) {
	o.emit(Event{Type: EventLog, Message: "Gerando plano com RAG..."})
	plan, err := o.planner.GeneratePlan(projectRoot, issueID, issue, "")
	if err != nil {
		return nil, err
	}
	o.emit(Event{Type: EventPlanGenerated, PlanID: plan.ID, Message: fmt.Sprintf("Plano com %d steps", len(plan.Steps)), Data: plan})
	if !dryRun {
		o.emit(Event{Type: EventApprovalRequired, PlanID: plan.ID, Message: "Aguardando aprovacao humana"})
	}
	return plan, nil
}

func (o *OrchestratorV2) ExecutePlan(plan *Plan, projectRoot string, approved bool) error {
	if !approved && !plan.Approved {
		return fmt.Errorf("plano nao aprovado")
	}
	if !plan.Approved {
		o.planner.ApprovePlan(plan)
	}
	for i, step := range plan.Steps {
		o.emit(Event{Type: EventStepStarted, PlanID: plan.ID, StepID: step.ID, Message: fmt.Sprintf("Step %d/%d: %s", i+1, len(plan.Steps), step.Title)})
		var err error
		start := time.Now()
		switch step.Type {
		case StepTypeGit:
			_, _, err = tools.EnsureBranch(projectRoot, plan.Project, plan.IssueID)
		case StepTypeTest:
			_, err = core.RunShellCommand(projectRoot, ".", "go test ./... 2>&1 | head -200")
		}
		o.audit.Log(AuditEntry{Project: plan.Project, IssueID: plan.IssueID, Tool: string(step.Type), Success: err == nil, DurationMs: time.Since(start).Milliseconds()})
		if err != nil {
			o.emit(Event{Type: EventFailed, PlanID: plan.ID, StepID: step.ID, Message: err.Error()})
			return err
		}
		o.emit(Event{Type: EventStepCompleted, PlanID: plan.ID, StepID: step.ID, Message: "Concluido: " + step.Title})
	}
	diff, _ := tools.GitDiff(projectRoot, false)
	o.emit(Event{Type: EventDiffReady, PlanID: plan.ID, Message: "Diff gerado", Data: diff})
	o.emit(Event{Type: EventCompleted, PlanID: plan.ID, Message: "Plano executado"})
	return nil
}
