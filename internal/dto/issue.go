package dto

// ===== Issue montada (base + projeto + tarefa) =====

type Request struct {
	Project   string      `json:"project,omitempty"`
	Issue     string      `json:"issue,omitempty"`
	Demanda   string      `json:"demanda"`
	Estrutura interface{} `json:"estrutura,omitempty"`
	Raiz      string      `json:"raiz"`
	Rules     []string    `json:"rules"`
	Arquivos  []string    `json:"arquivos"`
	Tarefas   []Tarefa    `json:"tarefas"`
	// Branch opcional: se preenchida, o auto-git usa este nome (ex: "dev-123").
	// Se vazia, fallback dsac/<projeto>/<issue-file>.
	Branch string `json:"branch,omitempty"`
}

type Tarefa struct {
	ID        string `json:"id"`
	Arquivo   string `json:"arquivo"`
	Tipo      string `json:"tipo"`
	Descricao string `json:"descricao"`
}
