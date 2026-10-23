package engineering

import (
	"os"
	"path/filepath"
	"strings"
)

// DetectStyle inspeciona a árvore do projeto e sugere um estilo.
func DetectStyle(projectRoot string) (Style, string) {
	if projectRoot == "" {
		return StyleLayers, "root vazio — default camadas"
	}
	hasPort := dirExists(filepath.Join(projectRoot, "internal", "port")) ||
		dirExists(filepath.Join(projectRoot, "internal", "ports"))
	hasAdapter := dirExists(filepath.Join(projectRoot, "internal", "adapter")) ||
		dirExists(filepath.Join(projectRoot, "internal", "adapters"))
	hasDomain := dirExists(filepath.Join(projectRoot, "internal", "domain"))
	hasModule := dirExists(filepath.Join(projectRoot, "internal", "module")) ||
		dirExists(filepath.Join(projectRoot, "internal", "modules"))
	hasService := dirExists(filepath.Join(projectRoot, "internal", "service")) ||
		dirExists(filepath.Join(projectRoot, "internal", "services"))
	hasRepo := dirExists(filepath.Join(projectRoot, "internal", "repository")) ||
		dirExists(filepath.Join(projectRoot, "internal", "repositories"))

	switch {
	case hasPort && hasAdapter:
		return StyleHexagonal, "encontrado internal/port(s) + adapter(s)"
	case hasDomain && !hasService:
		return StyleDDDLight, "encontrado internal/domain"
	case hasModule:
		return StyleModularMonolith, "encontrado internal/module(s)"
	case hasService || hasRepo:
		return StyleLayers, "encontrado service/repository (camadas)"
	default:
		return StyleLayers, "estrutura não identificada — default camadas"
	}
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

// ListTopInternal lista subpastas de internal/ (ajuda o wizard).
func ListTopInternal(projectRoot string) []string {
	dir := filepath.Join(projectRoot, "internal")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			out = append(out, "internal/"+e.Name())
		}
	}
	return out
}
