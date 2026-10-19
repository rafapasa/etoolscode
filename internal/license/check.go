package license

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const defaultAuthURL = "https://dsac.etoolstec.com.br/auth"

// Check chama o endpoint de licença.
// Bloqueia APENAS se HTTP 200 e mensagem indicar "acesso negado".
// Qualquer outro status, erro de rede, timeout ou body inesperado → permite (fail-open).
func Check(etoolsKey string) error {
	if strings.EqualFold(os.Getenv("ETOOLSCODE_LICENSE_CHECK"), "off") {
		return nil
	}
	url := strings.TrimSpace(os.Getenv("ETOOLSCODE_AUTH_URL"))
	if url == "" {
		url = defaultAuthURL
	}
	key := strings.TrimSpace(etoolsKey)
	if key == "" {
		key = strings.TrimSpace(os.Getenv("ETOOLSCODE_KEY"))
	}
	// sem key ainda (1ª execução) — não bloqueia
	if key == "" {
		return nil
	}

	body, _ := json.Marshal(map[string]string{"etoolscodekey": key})
	client := &http.Client{Timeout: 4 * time.Second}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		// endpoint inexistente / rede — deixa passar
		return nil
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	// tenta JSON { "message": "..." } ou { "msg": "..." } ou { "error": "..." }
	var m map[string]interface{}
	if json.Unmarshal(raw, &m) != nil {
		// 200 sem JSON legível — não bloqueia
		return nil
	}
	msg := strings.ToLower(fmt.Sprint(m["message"], " ", m["msg"], " ", m["error"], " ", m["status"]))
	if strings.Contains(msg, "acesso negado") {
		return fmt.Errorf("acesso negado pelo servidor de licença")
	}
	return nil
}
