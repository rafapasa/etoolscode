package engineering

import (
	"fmt"
	"strings"
	"unicode"
)

// BuildPlan gera um plano determinístico a partir do brief (sem LLM).
// O engenheiro revisa antes do scaffold.
func BuildPlan(b Brief) (*Plan, error) {
	b.Feature = strings.TrimSpace(b.Feature)
	b.PackagePath = strings.Trim(strings.TrimSpace(b.PackagePath), "/")
	b.Description = strings.TrimSpace(b.Description)
	if b.Feature == "" {
		return nil, fmt.Errorf("feature obrigatória")
	}
	if !ValidStyle(string(b.Style)) {
		return nil, fmt.Errorf("style inválido: %s", b.Style)
	}
	if b.PackagePath == "" {
		b.PackagePath = defaultPackage(b.Style, b.Feature)
	}
	pkg := packageName(b.PackagePath)
	iface := exportName(b.Feature)
	if !strings.HasSuffix(iface, "Service") && b.Style == StyleLayers {
		// em camadas costuma ser XService
		if !strings.Contains(iface, "Service") && !strings.Contains(iface, "Repository") {
			iface = iface + "Service"
		}
	}

	ifaceFile, testFile, stubFile := pathsFor(b.Style, b.PackagePath, iface, pkg)

	methods := defaultMethods(b.Style, iface, b.Description)

	p := &Plan{
		Style:       b.Style,
		Feature:     b.Feature,
		PackagePath: b.PackagePath,
		ModulePath:  strings.TrimSpace(b.ModulePath),
		Branch:      strings.TrimSpace(b.Branch),
		Summary:     fmt.Sprintf("[%s] %s — %s", b.Style, iface, b.Description),
		Folders:     foldersFor(b.Style, b.PackagePath),
		Interfaces: []Interface{{
			Name:        iface,
			File:        ifaceFile,
			Package:     pkg,
			Methods:     methods,
			TestFile:    testFile,
			StubFile:    stubFile,
			Description: b.Description,
		}},
		TestFiles: []string{testFile},
		Rules: []string{
			"TDD: implementar até os testes do scaffold passarem.",
			"Não alterar a assinatura das interfaces geradas sem atualizar os testes.",
			"Não expandir escopo além da descrição da feature.",
		},
		DoNot: []string{
			"Não gerar implementação completa nesta fase (apenas interface + testes + stub).",
			"Não criar DTOs de topo desnecessários.",
			"Não alterar arquivos fora da lista do plano.",
		},
	}
	return p, nil
}

func defaultPackage(style Style, feature string) string {
	base := strings.ToLower(exportName(feature))
	base = strings.TrimSuffix(base, "service")
	switch style {
	case StyleHexagonal:
		return "internal/port"
	case StyleDDDLight:
		return "internal/domain/" + base
	case StyleModularMonolith:
		return "internal/module/" + base
	default:
		return "internal/service"
	}
}

func pathsFor(style Style, pkgPath, iface, pkg string) (ifaceFile, testFile, stubFile string) {
	switch style {
	case StyleHexagonal:
		ifaceFile = strings.TrimSuffix(pkgPath, "/") + "/" + toSnake(iface) + ".go"
		// implementação no adapter
		stubFile = "internal/adapter/" + toSnake(iface) + ".go"
		testFile = "internal/adapter/" + toSnake(iface) + "_test.go"
	default:
		ifaceFile = strings.TrimSuffix(pkgPath, "/") + "/" + toSnake(iface) + ".go"
		// interface + stub no mesmo package; arquivo _iface e implementação
		// Para Go idiomático: interface no mesmo arquivo ou interface.go
		ifaceFile = strings.TrimSuffix(pkgPath, "/") + "/interface_" + toSnake(iface) + ".go"
		stubFile = strings.TrimSuffix(pkgPath, "/") + "/" + toSnake(iface) + ".go"
		testFile = strings.TrimSuffix(pkgPath, "/") + "/" + toSnake(iface) + "_test.go"
	}
	return ifaceFile, testFile, stubFile
}

func foldersFor(style Style, pkgPath string) []string {
	set := map[string]struct{}{pkgPath: {}}
	switch style {
	case StyleHexagonal:
		set["internal/port"] = struct{}{}
		set["internal/adapter"] = struct{}{}
	case StyleDDDLight:
		set[pkgPath] = struct{}{}
	case StyleModularMonolith:
		set[pkgPath] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out
}

func defaultMethods(style Style, iface, desc string) []Method {
	// Métodos genéricos de serviço; o dev ajusta no plano se quiser.
	_ = style
	_ = desc
	return []Method{
		{Name: "Execute", Signature: "ctx context.Context, input string) (string, error)", Doc: "Caso de uso principal da feature."},
	}
}

func packageName(pkgPath string) string {
	parts := strings.Split(pkgPath, "/")
	name := parts[len(parts)-1]
	if name == "" && len(parts) > 1 {
		name = parts[len(parts)-2]
	}
	// Go package identifiers
	var b strings.Builder
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "service"
	}
	return b.String()
}

func exportName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "Feature"
	}
	// se já parece PascalCase
	parts := splitIdent(s)
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
	}
	return strings.Join(parts, "")
}

func splitIdent(s string) []string {
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	// camelCase split
	var parts []string
	var cur strings.Builder
	for i, r := range s {
		if r == ' ' {
			if cur.Len() > 0 {
				parts = append(parts, cur.String())
				cur.Reset()
			}
			continue
		}
		if i > 0 && unicode.IsUpper(r) && cur.Len() > 0 {
			parts = append(parts, cur.String())
			cur.Reset()
		}
		cur.WriteRune(r)
	}
	if cur.Len() > 0 {
		parts = append(parts, cur.String())
	}
	return parts
}

func toSnake(s string) string {
	parts := splitIdent(s)
	for i, p := range parts {
		parts[i] = strings.ToLower(p)
	}
	return strings.Join(parts, "_")
}
