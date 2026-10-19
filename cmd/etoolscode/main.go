package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rafapasa/etoolscode/cmd/ui"
	"github.com/rafapasa/etoolscode/internal/config"
	"github.com/rafapasa/etoolscode/internal/core"
	"github.com/rafapasa/etoolscode/internal/dto"
	"github.com/rafapasa/etoolscode/internal/license"
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

	// Licença (fail-open): só bloqueia se servidor responder 200 + "acesso negado"
	licenseKey := os.Getenv("ETOOLSCODE_KEY")
	if licenseKey == "" {
		licenseKey = cfg.ResolveAPIKey()
	}
	if err := license.Check(licenseKey); err != nil {
		fmt.Printf("❌ %v\n", err)
		os.Exit(1)
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
	// Se a issue foi criada pela UI colando o JSON inteiro em "demanda", desembrulha.
	tarefa = unwrapNestedIssue(tarefa)

	var out dto.Request
	out.Raiz = preferRaiz(toString(tarefa["raiz"]), toString(projeto["raiz"]), toString(base["raiz"]))
	out.Demanda = toString(tarefa["demanda"])
	out.Arquivos = filterDummy(toStringSlice(tarefa["arquivos"]))
	out.Rules = append(out.Rules, toStringSlice(base["rules"])...)
	out.Rules = append(out.Rules, toStringSlice(projeto["rules"])...)
	out.Rules = append(out.Rules, toStringSlice(tarefa["rules"])...)
	out.Rules = append(out.Rules, toStringSlice(tarefa["regras_gerais"])...)
	if tarefa["estrutura"] != nil {
		out.Estrutura = tarefa["estrutura"]
	} else {
		out.Estrutura = projeto["estrutura"]
	}
	out.Tarefas = filterDummyTarefas(toTarefas(tarefa["tarefas"]))
	if out.Raiz == "" {
		return out, fmt.Errorf("raiz ausente (defina em projeto.json ou na issue)")
	}
	if !filepath.IsAbs(out.Raiz) {
		abs, err := filepath.Abs(out.Raiz)
		if err == nil {
			out.Raiz = abs
		}
	}
	if out.Demanda == "" {
		return out, fmt.Errorf("demanda ausente")
	}
	if len(out.Arquivos) == 0 {
		return out, fmt.Errorf("arquivos ausentes (lista fechada obrigatória)")
	}
	if len(out.Tarefas) == 0 {
		return out, fmt.Errorf("tarefas ausentes")
	}
	return out, nil
}

// unwrapNestedIssue: demanda pode ser um JSON stringificado da issue completa.
func unwrapNestedIssue(tarefa map[string]interface{}) map[string]interface{} {
	if tarefa == nil {
		return map[string]interface{}{}
	}
	dem := strings.TrimSpace(toString(tarefa["demanda"]))
	if dem == "" || dem[0] != '{' {
		return tarefa
	}
	var inner map[string]interface{}
	if err := json.Unmarshal([]byte(dem), &inner); err != nil {
		return tarefa
	}
	// só unwrap se parecer issue (tem demanda string ou tarefas/arquivos)
	if toString(inner["demanda"]) == "" && inner["tarefas"] == nil && inner["arquivos"] == nil {
		return tarefa
	}
	// mescla: campos internos prevalecem; mantém outer só se inner não tiver
	out := map[string]interface{}{}
	for k, v := range tarefa {
		out[k] = v
	}
	for k, v := range inner {
		out[k] = v
	}
	return out
}

func preferRaiz(vals ...string) string {
	for _, s := range vals {
		s = strings.TrimSpace(s)
		if s == "" || s == "." {
			continue // "." da UI create é lixo
		}
		return s
	}
	return ""
}

func filterDummy(in []string) []string {
	var out []string
	for _, s := range in {
		if s == "" || strings.EqualFold(s, "dummy") {
			continue
		}
		out = append(out, s)
	}
	return out
}

func filterDummyTarefas(in []dto.Tarefa) []dto.Tarefa {
	var out []dto.Tarefa
	for _, t := range in {
		if strings.EqualFold(t.Arquivo, "dummy") || strings.EqualFold(t.Descricao, "dummy") {
			continue
		}
		out = append(out, t)
	}
	return out
}

func toTarefas(v interface{}) []dto.Tarefa {
	raw, _ := v.([]interface{})
	var out []dto.Tarefa
	for _, item := range raw {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		t := dto.Tarefa{
			ID:        toString(m["id"]),
			Arquivo:   toString(m["arquivo"]),
			Tipo:      toString(m["tipo"]),
			Descricao: toString(m["descricao"]),
		}
		out = append(out, t)
	}
	return out
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
