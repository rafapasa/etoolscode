package engineering

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rafapasa/etoolscode/internal/core"
	"github.com/rafapasa/etoolscode/internal/dto"
)

// EnrichPlanWithLLM pede à LLM métodos mais realistas a partir da descrição.
// Em falha, devolve o plano original sem erro fatal.
func EnrichPlanWithLLM(llm core.LlmInterface, plan *Plan, description string) (*Plan, error) {
	if llm == nil || plan == nil || len(plan.Interfaces) == 0 {
		return plan, nil
	}
	iface := plan.Interfaces[0]
	prompt := fmt.Sprintf(`Você é um arquiteto Go sênior.
Dado o estilo %s e a interface %s no package %s,
proponha de 1 a 4 métodos Go realistas para: %s

Responda APENAS JSON válido no formato:
{"methods":[{"name":"Foo","signature":"ctx context.Context, id string) (string, error)","doc":"..."}]}

Regras:
- signature SEM o nome do método e SEM a palavra func; comece em ctx ou nos params.
- Inclua context.Context como primeiro parâmetro quando fizer sentido.
- Não invente dependências de frameworks.
`, plan.Style, iface.Name, iface.Package, description)

	msgs := []dto.Message{
		{Role: "system", Content: "Responda só JSON. Sem markdown."},
		{Role: "user", Content: prompt},
	}
	resp, err := llm.Chat(msgs, nil)
	if err != nil {
		return plan, fmt.Errorf("llm: %w", err)
	}
	content := ""
	if resp != nil && len(resp.Choices) > 0 {
		content = strings.TrimSpace(resp.Choices[0].Message.Content)
	}
	content = stripCodeFence(content)
	var parsed struct {
		Methods []Method `json:"methods"`
	}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil || len(parsed.Methods) == 0 {
		return plan, fmt.Errorf("parse métodos: %v", err)
	}
	// normaliza signatures
	for i := range parsed.Methods {
		parsed.Methods[i].Name = strings.TrimSpace(parsed.Methods[i].Name)
		sig := strings.TrimSpace(parsed.Methods[i].Signature)
		sig = strings.TrimPrefix(sig, "func")
		sig = strings.TrimSpace(sig)
		if strings.HasPrefix(sig, parsed.Methods[i].Name) {
			sig = strings.TrimSpace(sig[len(parsed.Methods[i].Name):])
		}
		if !strings.HasPrefix(sig, "(") {
			sig = "(" + strings.TrimPrefix(sig, "(")
		}
		// scaffold espera signature sem '(' inicial opcional — planner usa TrimPrefix "("
		parsed.Methods[i].Signature = strings.TrimPrefix(sig, "(")
	}
	plan.Interfaces[0].Methods = parsed.Methods
	plan.Summary = fmt.Sprintf("[%s] %s — %s (métodos enriquecidos por LLM)", plan.Style, iface.Name, description)
	return plan, nil
}

func stripCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```JSON")
		s = strings.TrimPrefix(s, "```")
		if i := strings.LastIndex(s, "```"); i >= 0 {
			s = s[:i]
		}
	}
	return strings.TrimSpace(s)
}
