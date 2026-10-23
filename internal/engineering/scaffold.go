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
	lang := plan.Language
	if lang == "" {
		lang = "go"
	}
	out := map[string]string{}
	for _, iface := range plan.Interfaces {
		out[iface.File] = renderInterfaceFile(lang, iface)
		out[iface.TestFile] = renderTestFile(lang, iface)
		out[iface.StubFile] = renderStubFile(lang, iface)
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

func renderInterfaceFile(lang string, iface Interface) string {
	switch lang {
	case "typescript":
		return tsInterface(iface)
	case "python":
		return pyInterface(iface)
	case "dart":
		return dartInterface(iface)
	case "java":
		return javaInterface(iface)
	case "csharp":
		return csInterface(iface)
	case "generic":
		return genericInterface(iface)
	default:
		return goInterface(iface)
	}
}

func renderStubFile(lang string, iface Interface) string {
	switch lang {
	case "typescript":
		return tsStub(iface)
	case "python":
		return pyStub(iface)
	case "dart":
		return dartStub(iface)
	case "java":
		return javaStub(iface)
	case "csharp":
		return csStub(iface)
	case "generic":
		return genericStub(iface)
	default:
		return goStub(iface)
	}
}

func renderTestFile(lang string, iface Interface) string {
	switch lang {
	case "typescript":
		return tsTest(iface)
	case "python":
		return pyTest(iface)
	case "dart":
		return dartTest(iface)
	case "java":
		return javaTest(iface)
	case "csharp":
		return csTest(iface)
	case "generic":
		return genericTest(iface)
	default:
		return goTest(iface)
	}
}

func goInterface(iface Interface) string {
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

func goStub(iface Interface) string {
	pkg := packageOf(iface, "stub")
	structName := implName(iface.Name)
	var b strings.Builder
	b.WriteString("package " + pkg + "\n\n")
	b.WriteString("import (\n\t\"context\"\n\t\"fmt\"\n)\n\n")
	b.WriteString("// " + structName + " esqueleto TDD (eToolsCode).\n")
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

func goTest(iface Interface) string {
	pkg := packageOf(iface, "test")
	structName := implName(iface.Name)
	var b strings.Builder
	b.WriteString("package " + pkg + "\n\n")
	b.WriteString("import (\n\t\"context\"\n\t\"strings\"\n\t\"testing\"\n)\n\n")
	for _, m := range iface.Methods {
		b.WriteString("func Test" + iface.Name + "_" + m.Name + "_AindaNaoImplementado(t *testing.T) {\n")
		b.WriteString("\tt.Parallel()\n")
		b.WriteString("\ts := &" + structName + "{}\n")
		b.WriteString("\t_, err := s." + m.Name + "(context.Background(), \"exemplo\")\n")
		b.WriteString("\tif err == nil {\n\t\tt.Fatal(\"esperava erro TODO\")\n\t}\n")
		b.WriteString("\tif !strings.Contains(strings.ToLower(err.Error()), \"todo\") {\n\t\tt.Fatalf(\"got %v\", err)\n\t}\n")
		b.WriteString("}\n\n")
	}
	return b.String()
}

func tsInterface(iface Interface) string {
	var b strings.Builder
	b.WriteString("/** " + iface.Name + " — " + sanitizeComment(iface.Description) + " */\n")
	b.WriteString("export interface " + iface.Name + " {\n")
	for _, m := range iface.Methods {
		b.WriteString("  /** " + sanitizeComment(m.Doc) + " */\n")
		b.WriteString("  " + strings.ToLower(m.Name[:1]) + m.Name[1:] + "(input: string): Promise<string>;\n")
	}
	b.WriteString("}\n")
	return b.String()
}

func tsStub(iface Interface) string {
	name := implName(iface.Name)
	var b strings.Builder
	b.WriteString("import type { " + iface.Name + " } from './" + strings.TrimSuffix(filepath.Base(iface.File), filepath.Ext(iface.File)) + "';\n\n")
	b.WriteString("/** Esqueleto TDD (eToolsCode) */\n")
	b.WriteString("export class " + name + " implements " + iface.Name + " {\n")
	for _, m := range iface.Methods {
		mn := strings.ToLower(m.Name[:1]) + m.Name[1:]
		b.WriteString("  async " + mn + "(input: string): Promise<string> {\n")
		b.WriteString("    void input;\n")
		b.WriteString("    throw new Error('TODO: implementar " + iface.Name + "." + m.Name + "');\n")
		b.WriteString("  }\n")
	}
	b.WriteString("}\n")
	return b.String()
}

func tsTest(iface Interface) string {
	name := implName(iface.Name)
	var b strings.Builder
	b.WriteString("import { " + name + " } from './" + strings.TrimSuffix(filepath.Base(iface.StubFile), filepath.Ext(iface.StubFile)) + "';\n\n")
	for _, m := range iface.Methods {
		mn := strings.ToLower(m.Name[:1]) + m.Name[1:]
		b.WriteString("test('" + iface.Name + "." + m.Name + " ainda nao implementado', async () => {\n")
		b.WriteString("  const s = new " + name + "();\n")
		b.WriteString("  await expect(s." + mn + "('exemplo')).rejects.toThrow(/TODO/i);\n")
		b.WriteString("});\n\n")
	}
	return b.String()
}

func pyInterface(iface Interface) string {
	var b strings.Builder
	b.WriteString("from typing import Protocol\n\n")
	b.WriteString("class " + iface.Name + "(Protocol):\n")
	b.WriteString("    \"\"\"" + sanitizeComment(iface.Description) + "\"\"\"\n")
	for _, m := range iface.Methods {
		mn := toSnake(m.Name)
		b.WriteString("    def " + mn + "(self, input: str) -> str:\n")
		b.WriteString("        ...\n")
	}
	return b.String()
}

func pyStub(iface Interface) string {
	name := implName(iface.Name)
	var b strings.Builder
	b.WriteString("class " + name + ":\n")
	b.WriteString("    \"\"\"Esqueleto TDD (eToolsCode).\"\"\"\n")
	for _, m := range iface.Methods {
		mn := toSnake(m.Name)
		b.WriteString("    def " + mn + "(self, input: str) -> str:\n")
		b.WriteString("        raise NotImplementedError('TODO: implementar " + iface.Name + "." + m.Name + "')\n")
	}
	return b.String()
}

func pyTest(iface Interface) string {
	name := implName(iface.Name)
	mod := strings.TrimSuffix(filepath.Base(iface.StubFile), ".py")
	var b strings.Builder
	b.WriteString("import pytest\nfrom " + mod + " import " + name + "\n\n")
	for _, m := range iface.Methods {
		mn := toSnake(m.Name)
		b.WriteString("def test_" + toSnake(iface.Name) + "_" + mn + "_not_implemented():\n")
		b.WriteString("    s = " + name + "()\n")
		b.WriteString("    with pytest.raises(NotImplementedError, match='TODO'):\n")
		b.WriteString("        s." + mn + "('exemplo')\n\n")
	}
	return b.String()
}

func dartInterface(iface Interface) string {
	var b strings.Builder
	b.WriteString("/// " + sanitizeComment(iface.Description) + "\n")
	b.WriteString("abstract class " + iface.Name + " {\n")
	for _, m := range iface.Methods {
		mn := strings.ToLower(m.Name[:1]) + m.Name[1:]
		b.WriteString("  Future<String> " + mn + "(String input);\n")
	}
	b.WriteString("}\n")
	return b.String()
}

func dartStub(iface Interface) string {
	name := implName(iface.Name)
	var b strings.Builder
	ifaceFile := strings.TrimSuffix(filepath.Base(iface.File), ".dart")
	b.WriteString("import '" + ifaceFile + ".dart';\n\n")
	b.WriteString("/// Esqueleto TDD (eToolsCode).\n")
	b.WriteString("class " + name + " implements " + iface.Name + " {\n")
	for _, m := range iface.Methods {
		mn := strings.ToLower(m.Name[:1]) + m.Name[1:]
		b.WriteString("  @override\n")
		b.WriteString("  Future<String> " + mn + "(String input) async {\n")
		b.WriteString("    throw UnimplementedError('TODO: implementar " + iface.Name + "." + m.Name + "');\n")
		b.WriteString("  }\n")
	}
	b.WriteString("}\n")
	return b.String()
}

func dartTest(iface Interface) string {
	name := implName(iface.Name)
	stub := strings.TrimSuffix(filepath.Base(iface.StubFile), ".dart")
	var b strings.Builder
	b.WriteString("import 'package:flutter_test/flutter_test.dart';\n")
	b.WriteString("import '" + stub + ".dart';\n\n")
	b.WriteString("void main() {\n")
	for _, m := range iface.Methods {
		mn := strings.ToLower(m.Name[:1]) + m.Name[1:]
		b.WriteString("  test('" + iface.Name + "." + m.Name + " ainda nao implementado', () async {\n")
		b.WriteString("    final s = " + name + "();\n")
		b.WriteString("    expect(() => s." + mn + "('exemplo'), throwsA(isA<UnimplementedError>()));\n")
		b.WriteString("  });\n")
	}
	b.WriteString("}\n")
	return b.String()
}

func javaInterface(iface Interface) string {
	var b strings.Builder
	b.WriteString("/** " + sanitizeComment(iface.Description) + " */\n")
	b.WriteString("public interface " + iface.Name + " {\n")
	for _, m := range iface.Methods {
		mn := strings.ToLower(m.Name[:1]) + m.Name[1:]
		b.WriteString("    String " + mn + "(String input) throws Exception;\n")
	}
	b.WriteString("}\n")
	return b.String()
}

func javaStub(iface Interface) string {
	name := implName(iface.Name)
	var b strings.Builder
	b.WriteString("/** Esqueleto TDD (eToolsCode). */\n")
	b.WriteString("public class " + name + " implements " + iface.Name + " {\n")
	for _, m := range iface.Methods {
		mn := strings.ToLower(m.Name[:1]) + m.Name[1:]
		b.WriteString("    @Override\n")
		b.WriteString("    public String " + mn + "(String input) throws Exception {\n")
		b.WriteString("        throw new UnsupportedOperationException(\"TODO: implementar " + iface.Name + "." + m.Name + "\");\n")
		b.WriteString("    }\n")
	}
	b.WriteString("}\n")
	return b.String()
}

func javaTest(iface Interface) string {
	name := implName(iface.Name)
	var b strings.Builder
	b.WriteString("import org.junit.jupiter.api.Test;\nimport static org.junit.jupiter.api.Assertions.*;\n\n")
	b.WriteString("class " + iface.Name + "Test {\n")
	for _, m := range iface.Methods {
		mn := strings.ToLower(m.Name[:1]) + m.Name[1:]
		b.WriteString("    @Test\n")
		b.WriteString("    void " + mn + "_notImplemented() {\n")
		b.WriteString("        " + name + " s = new " + name + "();\n")
		b.WriteString("        assertThrows(UnsupportedOperationException.class, () -> s." + mn + "(\"exemplo\"));\n")
		b.WriteString("    }\n")
	}
	b.WriteString("}\n")
	return b.String()
}

func csInterface(iface Interface) string {
	var b strings.Builder
	b.WriteString("namespace EtoolsCode.Generated;\n\n")
	b.WriteString("/// <summary>" + sanitizeComment(iface.Description) + "</summary>\n")
	b.WriteString("public interface " + iface.Name + "\n{\n")
	for _, m := range iface.Methods {
		b.WriteString("    Task<string> " + m.Name + "Async(string input);\n")
	}
	b.WriteString("}\n")
	return b.String()
}

func csStub(iface Interface) string {
	name := implName(iface.Name)
	var b strings.Builder
	b.WriteString("namespace EtoolsCode.Generated;\n\n")
	b.WriteString("/// <summary>Esqueleto TDD (eToolsCode).</summary>\n")
	b.WriteString("public class " + name + " : " + iface.Name + "\n{\n")
	for _, m := range iface.Methods {
		b.WriteString("    public Task<string> " + m.Name + "Async(string input)\n")
		b.WriteString("        => throw new NotImplementedException(\"TODO: implementar " + iface.Name + "." + m.Name + "\");\n")
	}
	b.WriteString("}\n")
	return b.String()
}

func csTest(iface Interface) string {
	name := implName(iface.Name)
	var b strings.Builder
	b.WriteString("using Xunit;\n\nnamespace EtoolsCode.Generated.Tests;\n\n")
	b.WriteString("public class " + iface.Name + "Tests\n{\n")
	for _, m := range iface.Methods {
		b.WriteString("    [Fact]\n")
		b.WriteString("    public async Task " + m.Name + "_NotImplemented()\n    {\n")
		b.WriteString("        var s = new " + name + "();\n")
		b.WriteString("        await Assert.ThrowsAsync<NotImplementedException>(() => s." + m.Name + "Async(\"exemplo\"));\n")
		b.WriteString("    }\n")
	}
	b.WriteString("}\n")
	return b.String()
}

func genericInterface(iface Interface) string {
	return "# Interface " + iface.Name + "\n# " + sanitizeComment(iface.Description) + "\n# Methods: " + strings.Join(methodNames(iface), ", ") + "\n"
}

func genericStub(iface Interface) string {
	return "# Stub " + implName(iface.Name) + "\n# TODO: implement " + strings.Join(methodNames(iface), ", ") + "\n"
}

func genericTest(iface Interface) string {
	return "# Tests for " + iface.Name + "\n# Assert each method still raises TODO until implemented\n"
}

func methodNames(iface Interface) []string {
	var out []string
	for _, m := range iface.Methods {
		out = append(out, m.Name)
	}
	return out
}

func sanitizeComment(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), "\n", " ")
}
