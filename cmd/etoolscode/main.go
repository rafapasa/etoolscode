package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rafapasa/etoolscode/cmd/ui"
	"github.com/rafapasa/etoolscode/internal/config"
	"github.com/rafapasa/etoolscode/internal/core"
	"github.com/rafapasa/etoolscode/internal/dto"
	"github.com/rafapasa/etoolscode/internal/service"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg := config.NewConfig()
	uiMode := flag.Bool("ui", false, "modo interface web")
	hashPassword := flag.String("hash-password", "", "gera bcrypt hash da senha e sai")
	flag.Parse()

	if *hashPassword != "" {
		h, err := bcrypt.GenerateFromPassword([]byte(*hashPassword), bcrypt.DefaultCost)
		if err != nil {
			fmt.Printf("erro hash: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(string(h))
		return
	}

	// Modo UI não exige API key: a chave só é necessária ao executar issues/chat LLM.
	// UI única (chat + issues + fontes + git).
	if *uiMode {
		if err := ui.Start(cfg); err != nil {
			fmt.Printf("❌ Erro: %v\n", err)
			os.Exit(1)
		}
		return
	}

	key := cfg.ResolveAPIKey()
	if key == "" {
		if c, err := ui.LoadConfig(); err == nil && c.DeepSeekApiKey != "" {
			key = c.DeepSeekApiKey
		}
	}
	if key == "" {
		fmt.Println("❌ API Key não encontrada")
		os.Exit(1)
	}

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Uso: --ui ou <issue.json>")
		os.Exit(1)
	}

	tarefaPath := args[0]
	basePath := os.Getenv("DS_AC_BASE")
	if basePath == "" {
		basePath = "/home/opc/prj/issues/base.json"
	}
	projetoPath := filepath.Join(filepath.Dir(tarefaPath), "projeto.json")

	base, _ := loadJSON(basePath)
	projeto, _ := loadJSON(projetoPath)
	tarefa, err := loadJSON(tarefaPath)
	if err != nil {
		fmt.Printf("❌ tarefa: %v\n", err)
		os.Exit(1)
	}

	req, err := merge(base, projeto, tarefa)
	if err != nil {
		fmt.Printf("❌ merge: %v\n", err)
		os.Exit(1)
	}
	// projeto/issue a partir do caminho da tarefa (issues/<projeto>/<issue>.json)
	req.Issue = filepath.Base(tarefaPath)
	req.Project = filepath.Base(filepath.Dir(tarefaPath))

	o := service.NewOrchestratorWithConfig(core.NewLlmClient(*cfg), cfg)
	if err := o.Start(req); err != nil {
		fmt.Printf("❌ Erro: %v\n", err)
		os.Exit(1)
	}
}

func loadJSON(path string) (map[string]interface{}, error) {
	abs, _ := filepath.Abs(path)
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	var out map[string]interface{}
	json.Unmarshal(data, &out)
	return out, nil
}

func merge(base, projeto, tarefa map[string]interface{}) (dto.Request, error) {
	var out dto.Request
	out.Raiz = firstNonEmpty(toString(tarefa["raiz"]), toString(projeto["raiz"]), toString(base["raiz"]))
	out.Demanda = toString(tarefa["demanda"])
	out.Arquivos = toStringSlice(tarefa["arquivos"])
	if out.Raiz == "" {
		return out, fmt.Errorf("raiz ausente")
	}
	if out.Demanda == "" {
		return out, fmt.Errorf("demanda ausente")
	}
	return out, nil
}

func toString(v interface{}) string { s, _ := v.(string); return s }

func toStringSlice(v interface{}) []string {
	raw, _ := v.([]interface{})
	var out []string
	for _, i := range raw {
		if s, ok := i.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}
