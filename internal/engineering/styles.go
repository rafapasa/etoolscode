package engineering

// Style estilos de arquitetura suportados pelo assistente.
type Style string

const (
	StyleLayers          Style = "layers"
	StyleHexagonal       Style = "hexagonal"
	StyleDDDLight        Style = "ddd_light"
	StyleModularMonolith Style = "modular_monolith"
)

type StyleInfo struct {
	ID          Style  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func AvailableStyles() []StyleInfo {
	return []StyleInfo{
		{StyleLayers, "Camadas", "handler → service → repository. Padrão clássico e próximo do stack atual eTools."},
		{StyleHexagonal, "Hexagonal (Ports & Adapters)", "Ports (interfaces) no domínio/aplicação; adapters externos. Ideal para TDD."},
		{StyleDDDLight, "DDD tático leve", "Aggregate, domain service e application service sem cerimônia excessiva."},
		{StyleModularMonolith, "Monólito modular", "Pacotes por feature/módulo com API interna clara entre módulos."},
	}
}

func ValidStyle(s string) bool {
	switch Style(s) {
	case StyleLayers, StyleHexagonal, StyleDDDLight, StyleModularMonolith:
		return true
	}
	return false
}
