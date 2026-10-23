package engineering

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GenerateFiles produz o conteúdo dos arquivos do scaffold (sem gravar).
func GenerateFiles(plan *Plan) (map[string]string, error) {
	if plan == nil || len(plan.Interfaces) == 0 {
		return nil, fmt.Errorf("plano sem interfaces")
	}
	out := map[string]string{}
	for _, iface := range plan.Interfaces {
		out[iface.File] = renderInterfaceFile(iface)
		out[iface.TestFile] = renderTestFile(iface)
		out[iface.StubFile] = renderStubFile(iface)
	}
	return out, nil
}

// WriteScaffold grava os arquivos sob projectRoot. skipExisting evita sobrescrever.
func WriteScaffold(projectRoot string, plan *Plan, skipExisting bool) (*ScaffoldResult, error) {
	files, err := GenerateFiles(plan)
	if err != nil {
		return nil, err
	}
	res := &ScaffoldResult{}
	for rel, content := range files {
		full := filepath.Join(projectRoot, filepath.FromSlash(rel))
		if skipExisting {
			if _, err := os.Stat(full); err == nil {
				res.Skipped = append(res.Skipped, rel)
				continue
			}
		}
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			return res, err
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			return res, err
		}
		res.Written = append(res.Written, rel)
	}
	return res, nil
}

func implName(ifaceName string) string {
	base := strings.TrimSuffix(ifaceName, "Service")
	if base == "" || base == ifaceName {
		return ifaceName + "Impl"
	}
	return base + "Impl"
}

func packageOf(iface Interface, which string) string {
	path := iface.File
	if which == "stub" {
		path = iface.StubFile
	} else if which == "test" {
		path = iface.TestFile
	}
	if strings.Contains(path, "/adapter/") || strings.HasPrefix(path, "internal/adapter") {
		return "adapter"
	}
	return iface.Package
}

func renderInterfaceFile(iface Interface) string {
	var b strings.Builder
	b.WriteString("package " + packageOf(iface, "iface") + "\n\n")
	b.WriteString("import \"context\"\n\n")
	if iface.Description != "" {
		b.WriteString("// " + iface.Name + " " + sanitizeComment(iface.Description) + "\n")
	}
	b.WriteString("type " + iface.Name + " interface {\n")
	for _, m := range iface.Methods {
		if m.Doc != "" {
			b.WriteString("\t// " + sanitizeComment(m.Doc) + "\n")
		}
		sig := strings.TrimPrefix(m.Signature, "(")
		b.WriteString("\t" + m.Name + "(" + sig + "\n")
	}
	b.WriteString("}\n")
	return b.String()
}

func renderStubFile(iface Interface) string {
	pkg := packageOf(iface, "stub")
	structName := implName(iface.Name)
	var b strings.Builder
	b.WriteString("package " + pkg + "\n\n")
	b.WriteString("import (\n\t\"context\"\n\t\"fmt\"\n)\n\n")
	b.WriteString("// " + structName + " esqueleto gerado pelo Assistente de Engenharia (TDD).\n")
	b.WriteString("// Implemente os métodos até os testes passarem.\n")
	b.WriteString("type " + structName + " struct{}\n\n")
	for _, m := range iface.Methods {
		sig := strings.TrimPrefix(m.Signature, "(")
		b.WriteString("func (s *" + structName + ") " + m.Name + "(" + sig + " {\n")
		b.WriteString("\t_ = ctx\n")
		b.WriteString("\treturn \"\", fmt.Errorf(\"TODO: implementar " + iface.Name + "." + m.Name + "\")\n")
		b.WriteString("}\n\n")
	}
	return b.String()
}

func renderTestFile(iface Interface) string {
	pkg := packageOf(iface, "test")
	structName := implName(iface.Name)
	var b strings.Builder
	b.WriteString("package " + pkg + "\n\n")
	b.WriteString("import (\n\t\"context\"\n\t\"strings\"\n\t\"testing\"\n)\n\n")
	b.WriteString("// Testes gerados pelo Assistente de Engenharia.\n")
	b.WriteString("// Test*AindaNaoImplementado garante estado vermelho inicial.\n")
	for _, m := range iface.Methods {
		b.WriteString("func Test" + iface.Name + "_" + m.Name + "_AindaNaoImplementado(t *testing.T) {\n")
		b.WriteString("\tt.Parallel()\n")
		b.WriteString("\ts := &" + structName + "{}\n")
		b.WriteString("\t_, err := s." + m.Name + "(context.Background(), \"exemplo\")\n")
		b.WriteString("\tif err == nil {\n")
		b.WriteString("\t\tt.Fatal(\"esperava erro TODO até implementar " + m.Name + "\")\n")
		b.WriteString("\t}\n")
		b.WriteString("\tif !strings.Contains(strings.ToLower(err.Error()), \"todo\") {\n")
		b.WriteString("\t\tt.Fatalf(\"erro deveria indicar TODO, got: %v\", err)\n")
		b.WriteString("\t}\n")
		b.WriteString("}\n\n")
	}
	return b.String()
}

func sanitizeComment(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), "\n", " ")
}
