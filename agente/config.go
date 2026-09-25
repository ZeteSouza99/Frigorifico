package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

type SerialConfig struct {
	Porta         string `json:"porta"`
	Baud          int    `json:"baud"`
	BitsDados     int    `json:"bits_dados"`
	Paridade      string `json:"paridade"` // "nenhuma", "par" ou "impar"
	BitsParada    int    `json:"bits_parada"`
	Protocolo     string `json:"protocolo"`      // "toledo_p03" ou "generico"
	CasasDecimais int    `json:"casas_decimais"` // -1 = automático (via status do toledo_p03)
}

type CapturaConfig struct {
	PesoMinimoKg     float64 `json:"peso_minimo_kg"`    // abaixo disso não registra (gancho vazio/ruído)
	LeiturasEstaveis int     `json:"leituras_estaveis"` // nº de leituras consecutivas para considerar estável
	ToleranciaKg     float64 `json:"tolerancia_kg"`     // variação máxima entre as leituras estáveis
	RearmeKg         float64 `json:"rearme_kg"`         // peso precisa cair abaixo disso p/ liberar nova pesagem
}

type APIConfig struct {
	URL   string `json:"url"`
	Chave string `json:"chave"`
}

type ImpressoraConfig struct {
	Tipo     string `json:"tipo"`     // "rede", "windows", "arquivo" ou "desativada"
	Endereco string `json:"endereco"` // rede: "192.168.0.50:9100" | windows: nome da impressora | arquivo: pasta
	Copias   int    `json:"copias"`
	Template string `json:"template"` // caminho do arquivo .zpl
}

type Config struct {
	Serial     SerialConfig     `json:"serial"`
	Captura    CapturaConfig    `json:"captura"`
	API        APIConfig        `json:"api"`
	Impressora ImpressoraConfig `json:"impressora"`
}

func configPadrao() Config {
	return Config{
		Serial: SerialConfig{
			Porta:         "COM3",
			Baud:          4800,
			BitsDados:     7,
			Paridade:      "par",
			BitsParada:    1,
			Protocolo:     "toledo_p03",
			CasasDecimais: 0,
		},
		Captura: CapturaConfig{
			PesoMinimoKg:     30,
			LeiturasEstaveis: 8,
			ToleranciaKg:     1.0,
			RearmeKg:         15,
		},
		API: APIConfig{
			URL:   "http://localhost:8080",
			Chave: "",
		},
		Impressora: ImpressoraConfig{
			Tipo:     "arquivo",
			Endereco: "etiquetas",
			Copias:   1,
			Template: "etiqueta.zpl",
		},
	}
}

// carregarConfig lê o arquivo de configuração; cria com valores padrão se não existir.
func carregarConfig(caminho string) (Config, bool, error) {
	dados, err := os.ReadFile(caminho)
	if os.IsNotExist(err) {
		cfg := configPadrao()
		b, _ := json.MarshalIndent(cfg, "", "  ")
		if err := os.WriteFile(caminho, b, 0o644); err != nil {
			return cfg, false, fmt.Errorf("erro ao criar %s: %w", caminho, err)
		}
		return cfg, true, nil
	}
	if err != nil {
		return Config{}, false, err
	}
	dados = bytes.TrimPrefix(dados, []byte{0xEF, 0xBB, 0xBF}) // remove BOM (Bloco de Notas)
	cfg := configPadrao()
	if err := json.Unmarshal(dados, &cfg); err != nil {
		return Config{}, false, fmt.Errorf("erro ao ler %s: %w", caminho, err)
	}
	return cfg, false, nil
}
