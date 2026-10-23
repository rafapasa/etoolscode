package engineering

// Brief entrada do wizard de engenharia.
type Brief struct {
	Project     string `json:"project"`
	Style       Style  `json:"style"`
	Language    string `json:"language"` // go|typescript|python|dart|java|csharp|generic
	Feature     string `json:"feature"`      // nome curto ex: TenantResolver
	PackagePath string `json:"package_path"` // ex: internal/service ou internal/tenant
	Description string `json:"description"`  // o que a feature deve fazer
	ModulePath  string `json:"module_path"`  // opcional: github.com/org/repo
	Branch      string `json:"branch"`       // opcional
}

// Plan plano revisável antes do scaffold TDD.
type Plan struct {
	Style       Style        `json:"style"`
	Language    string       `json:"language"`
	Feature     string       `json:"feature"`
	PackagePath string       `json:"package_path"`
	ModulePath  string       `json:"module_path,omitempty"`
	Branch      string       `json:"branch,omitempty"`
	Summary     string       `json:"summary"`
	Folders     []string     `json:"folders"`
	Interfaces  []Interface  `json:"interfaces"`
	TestFiles   []string     `json:"test_files"`
	Rules       []string     `json:"rules"`
	DoNot       []string     `json:"do_not"`
}

type Interface struct {
	Name        string   `json:"name"`
	File        string   `json:"file"`         // path relativo no projeto
	Package     string   `json:"package"`      // nome do package go
	Methods     []Method `json:"methods"`
	TestFile    string   `json:"test_file"`
	StubFile    string   `json:"stub_file"`    // implementação com panic/TODO
	Description string   `json:"description"`
}

type Method struct {
	Name       string `json:"name"`
	Signature  string `json:"signature"`  // sem o nome do receiver/interface
	Doc        string `json:"doc,omitempty"`
}

// ScaffoldResult arquivos gerados no disco + issue sugerida.
type ScaffoldResult struct {
	Written []string `json:"written"`
	Skipped []string `json:"skipped"` // já existiam
	Issue   string   `json:"issue,omitempty"` // path da issue criada se solicitado
}
