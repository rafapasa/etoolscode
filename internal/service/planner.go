package service

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rafapasa/etoolscode/internal/dto"
)

type StepType string

const (
	StepTypeRead  StepType = "read"
	StepTypeWrite StepType = "write"
	StepTypeTest  StepType = "test"
	StepTypeGit   StepType = "git"
)

type StepStatus string

const (
	StepStatusPending  StepStatus = "pending"
	StepStatusApproved StepStatus = "approved"
	StepStatusDoing    StepStatus = "doing"
	StepStatusDone     StepStatus = "done"
	StepStatusFailed   StepStatus = "failed"
)

type ToolCallSpec struct {
	Tool    string         `json:"tool"`
	Args    map[string]any `json:"args,omitempty"`
	Purpose string         `json:"purpose,omitempty"`
}

type PlanStep struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Files       []string       `json:"files"`
	Type        StepType       `json:"type"`
	Status      StepStatus     `json:"status"`
	ToolCalls   []ToolCallSpec `json:"tool_calls"`
	Estimated   string         `json:"estimated,omitempty"`
}

type Plan struct {
	ID         string     `json:"id"`
	IssueID    string     `json:"issue_id"`
	Project    string     `json:"project"`
	Goal       string     `json:"goal"`
	Steps      []PlanStep `json:"steps"`
	CreatedAt  time.Time  `json:"created_at"`
	Approved   bool       `json:"approved"`
	RAGContext string     `json:"rag_context,omitempty"`
}

type PlannerService struct {
	rag *RAGService
}

func NewPlannerService(rag *RAGService) *PlannerService {
	if rag == nil {
		rag = NewRAGService("")
	}
	return &PlannerService{rag: rag}
}

func (p *PlannerService) GeneratePlan(projectRoot, issueID string, issue dto.Request, extraContext string) (*Plan, error) {
	goal := issue.Demanda
	if goal == "" {
		goal = "Executar tarefa"
	}
	ragCtx := ""
	if p.rag != nil && projectRoot != "" {
		_ = p.rag.BuildIndex(projectRoot)
		ragCtx = p.rag.GetRelevantContext(goal, 2500)
	}
	plan := &Plan{
		ID:         uuid.New().String(),
		IssueID:    issueID,
		Project:    projectRoot,
		Goal:       goal,
		CreatedAt:  time.Now(),
		RAGContext: ragCtx,
	}
	plan.Steps = p.heuristicBreakdown(issue)
	return plan, nil
}

func (p *PlannerService) heuristicBreakdown(issue dto.Request) []PlanStep {
	var steps []PlanStep
	idGen := func() string { return uuid.New().String()[:8] }
	steps = append(steps, PlanStep{ID: idGen(), Title: "Analise do codebase", Description: "Ler arquivos relevantes", Type: StepTypeRead, Status: StepStatusPending, Estimated: "1-2 min"})
	steps = append(steps, PlanStep{ID: idGen(), Title: "Criar branch seguro", Description: "Branch etoolcode/<issue>", Type: StepTypeGit, Status: StepStatusPending, Estimated: "10s"})
	for _, f := range issue.Arquivos {
		if f == "" {
			continue
		}
		steps = append(steps, PlanStep{ID: idGen(), Title: fmt.Sprintf("Implementar %s", f), Description: f, Files: []string{f}, Type: StepTypeWrite, Status: StepStatusPending, Estimated: "2-5 min"})
	}
	if len(issue.Arquivos) == 0 {
		steps = append(steps, PlanStep{ID: idGen(), Title: "Implementar mudanças", Description: issue.Demanda, Type: StepTypeWrite, Status: StepStatusPending})
	}
	steps = append(steps, PlanStep{ID: idGen(), Title: "Testes", Description: "go test", Type: StepTypeTest, Status: StepStatusPending})
	steps = append(steps, PlanStep{ID: idGen(), Title: "Diff para aprovacao", Description: "git diff", Type: StepTypeGit, Status: StepStatusPending})
	return steps
}

func (p *PlannerService) ApprovePlan(plan *Plan) {
	plan.Approved = true
	for i := range plan.Steps {
		if plan.Steps[i].Status == StepStatusPending {
			plan.Steps[i].Status = StepStatusApproved
		}
	}
}
