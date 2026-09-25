package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const templatePadraoZPL = `^XA
^CI28
^PW812
^LL406
^LH0,0
^CF0,38
^FO30,25^FDPESAGEM DE GADO - FRIGORIFICO^FS
^FO30,72^GB752,3,3^FS
^CF0,100
^FO30,105^FD{PESO} kg^FS
^CF0,40
^FO560,125^FD{ARROBAS} @^FS
^CF0,28
^FO30,225^FDDATA: {DATA}    HORA: {HORA}^FS
^FO30,262^FDBRINCO: {BRINCO}^FS
^BY2,2,80
^FO420,225^BCN,80,Y,N,N^FD{CODIGO}^FS
^XZ
`

// DadosEtiqueta são os campos disponíveis no template ZPL.
type DadosEtiqueta struct {
	PesoKg float64
	Brinco string
	Codigo string
	Quando time.Time
}

func renderizarEtiqueta(template string, d DadosEtiqueta) string {
	arrobas := d.PesoKg * 0.5 / 15 // rendimento de carcaça 50%
	brinco := d.Brinco
	if brinco == "" {
		brinco = "-"
	}
	sub := strings.NewReplacer(
		"{PESO}", strings.Replace(fmt.Sprintf("%.1f", d.PesoKg), ".", ",", 1),
		"{ARROBAS}", strings.Replace(fmt.Sprintf("%.1f", arrobas), ".", ",", 1),
		"{DATA}", d.Quando.Format("02/01/2006"),
		"{HORA}", d.Quando.Format("15:04:05"),
		"{CODIGO}", d.Codigo,
		"{BRINCO}", brinco,
	)
	return sub.Replace(template)
}

// carregarTemplate lê o template ZPL; cria o arquivo padrão se não existir.
func carregarTemplate(caminho string) (string, error) {
	b, err := os.ReadFile(caminho)
	if os.IsNotExist(err) {
		if err := os.WriteFile(caminho, []byte(templatePadraoZPL), 0o644); err != nil {
			return "", err
		}
		return templatePadraoZPL, nil
	}
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Imprimir envia o ZPL para a impressora Zebra conforme a configuração.
func Imprimir(cfg ImpressoraConfig, zpl string, codigo string) error {
	copias := cfg.Copias
	if copias < 1 {
		copias = 1
	}
	switch strings.ToLower(cfg.Tipo) {
	case "desativada", "":
		return nil
	case "rede":
		return imprimirRede(cfg.Endereco, zpl, copias)
	case "windows":
		return imprimirWindows(cfg.Endereco, zpl, copias)
	case "arquivo":
		return imprimirArquivo(cfg.Endereco, zpl, codigo)
	default:
		return fmt.Errorf("tipo de impressora desconhecido: %s", cfg.Tipo)
	}
}

// imprimirRede envia via TCP porta raw (JetDirect, padrão Zebra: 9100).
func imprimirRede(endereco, zpl string, copias int) error {
	if !strings.Contains(endereco, ":") {
		endereco += ":9100"
	}
	conn, err := net.DialTimeout("tcp", endereco, 5*time.Second)
	if err != nil {
		return fmt.Errorf("erro ao conectar na impressora %s: %w", endereco, err)
	}
	defer conn.Close()
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	for i := 0; i < copias; i++ {
		if _, err := conn.Write([]byte(zpl)); err != nil {
			return err
		}
	}
	return nil
}

// imprimirArquivo grava o ZPL em uma pasta (útil para testes e conferência
// visual em https://labelary.com/viewer.html).
func imprimirArquivo(pasta, zpl, codigo string) error {
	if pasta == "" {
		pasta = "etiquetas"
	}
	if err := os.MkdirAll(pasta, 0o755); err != nil {
		return err
	}
	nome := filepath.Join(pasta, "etiqueta-"+codigo+".zpl")
	return os.WriteFile(nome, []byte(zpl), 0o644)
}
