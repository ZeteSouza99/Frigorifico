package main

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
)

// Leitura é um valor de peso extraído do fluxo serial.
type Leitura struct {
	PesoKg          float64
	EmMovimento     bool
	MovimentoValido bool // true quando o protocolo informa o status de movimento
}

// Parser converte bytes crus da serial em leituras de peso.
type Parser interface {
	Alimentar(dados []byte) []Leitura
}

func novoParser(cfg SerialConfig) Parser {
	if strings.EqualFold(cfg.Protocolo, "generico") {
		return &ParserGenerico{}
	}
	return &ParserToledoP03{casas: cfg.CasasDecimais}
}

// ParserToledoP03 lê o protocolo contínuo Toledo (P03):
// STX(0x02) | SWA | SWB | SWC | peso 6 dígitos | [tara 6 dígitos] | CR(0x0D)
type ParserToledoP03 struct {
	buf   bytes.Buffer
	casas int
}

func (p *ParserToledoP03) Alimentar(dados []byte) []Leitura {
	p.buf.Write(dados)
	var out []Leitura
	for {
		b := p.buf.Bytes()
		ini := bytes.IndexByte(b, 0x02)
		if ini < 0 {
			p.buf.Reset()
			return out
		}
		fim := bytes.IndexByte(b[ini:], 0x0D)
		if fim < 0 {
			// frame incompleto: preserva do STX em diante
			resto := append([]byte(nil), b[ini:]...)
			p.buf.Reset()
			p.buf.Write(resto)
			return out
		}
		frame := b[ini : ini+fim]
		resto := append([]byte(nil), b[ini+fim+1:]...)
		p.buf.Reset()
		p.buf.Write(resto)

		if l, ok := p.decodificar(frame); ok {
			out = append(out, l)
		}
	}
}

func (p *ParserToledoP03) decodificar(frame []byte) (Leitura, bool) {
	// frame = STX + 3 bytes de status + 6 dígitos de peso (+ extras)
	if len(frame) < 10 {
		return Leitura{}, false
	}
	swa, swb := frame[1], frame[2]
	campo := string(frame[4:10])
	valor, err := strconv.Atoi(strings.TrimSpace(campo))
	if err != nil {
		return Leitura{}, false
	}

	casas := p.casas
	if casas < 0 { // automático: bits 0-2 do SWA (padrão Toledo: 2=1 casa, 3=2 casas, 4=3 casas)
		switch swa & 0x07 {
		case 2:
			casas = 1
		case 3:
			casas = 2
		case 4:
			casas = 3
		default:
			casas = 0
		}
	}
	peso := float64(valor)
	for i := 0; i < casas; i++ {
		peso /= 10
	}
	if swb&0x02 != 0 { // bit 1 = peso negativo
		peso = -peso
	}
	return Leitura{
		PesoKg:          peso,
		EmMovimento:     swb&0x08 != 0, // bit 3 = balança em movimento
		MovimentoValido: true,
	}, true
}

// ParserGenerico extrai o primeiro número de cada linha terminada em CR/LF.
// Funciona com indicadores configurados para enviar texto simples, ex.: "  472.5 kg".
type ParserGenerico struct {
	buf bytes.Buffer
}

var reNumero = regexp.MustCompile(`-?\d+(?:[.,]\d+)?`)

func (p *ParserGenerico) Alimentar(dados []byte) []Leitura {
	p.buf.Write(dados)
	var out []Leitura
	for {
		b := p.buf.Bytes()
		fim := bytes.IndexAny(b, "\r\n")
		if fim < 0 {
			if p.buf.Len() > 4096 {
				p.buf.Reset()
			}
			return out
		}
		linha := strings.TrimSpace(string(b[:fim]))
		resto := append([]byte(nil), b[fim+1:]...)
		p.buf.Reset()
		p.buf.Write(resto)

		if linha == "" {
			continue
		}
		m := reNumero.FindString(linha)
		if m == "" {
			continue
		}
		peso, err := strconv.ParseFloat(strings.Replace(m, ",", ".", 1), 64)
		if err != nil {
			continue
		}
		out = append(out, Leitura{PesoKg: peso})
	}
}
