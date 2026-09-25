package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"go.bug.st/serial"
)

func abrirSerial(cfg SerialConfig) (serial.Port, error) {
	modo := &serial.Mode{
		BaudRate: cfg.Baud,
		DataBits: cfg.BitsDados,
		StopBits: serial.OneStopBit,
		Parity:   serial.NoParity,
	}
	switch strings.ToLower(cfg.Paridade) {
	case "par", "even":
		modo.Parity = serial.EvenParity
	case "impar", "ímpar", "odd":
		modo.Parity = serial.OddParity
	}
	if cfg.BitsParada == 2 {
		modo.StopBits = serial.TwoStopBits
	}
	porta, err := serial.Open(cfg.Porta, modo)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir %s: %w (use -listar para ver as portas disponíveis)", cfg.Porta, err)
	}
	porta.SetReadTimeout(500 * time.Millisecond)
	return porta, nil
}

func listarPortas() {
	portas, err := serial.GetPortsList()
	if err != nil {
		log.Fatalf("erro ao listar portas: %v", err)
	}
	if len(portas) == 0 {
		fmt.Println("Nenhuma porta serial encontrada.")
		fmt.Println("Dica: conecte o cabo da balança (ou o adaptador USB-RS232) e rode novamente.")
		return
	}
	fmt.Println("Portas seriais disponíveis:")
	for _, p := range portas {
		fmt.Println("  -", p)
	}
}

// lerSerial lê continuamente a porta e publica leituras no canal.
// Reabre a porta automaticamente em caso de erro (cabo desconectado etc).
func lerSerial(cfg SerialConfig, leituras chan<- Leitura) {
	falhas := 0
	for {
		porta, err := abrirSerial(cfg)
		if err != nil {
			// loga a 1ª falha e depois 1x por minuto para não inflar o log
			if falhas%12 == 0 {
				log.Printf("ERRO serial: %v — tentando reconectar a cada 5s", err)
			}
			falhas++
			time.Sleep(5 * time.Second)
			continue
		}
		falhas = 0
		log.Printf("porta %s aberta (%d baud, %d%s%d, protocolo %s)",
			cfg.Porta, cfg.Baud, cfg.BitsDados, strings.ToUpper(cfg.Paridade[:1]), cfg.BitsParada, cfg.Protocolo)

		parser := novoParser(cfg)
		buf := make([]byte, 256)
		for {
			n, err := porta.Read(buf)
			if err != nil {
				log.Printf("ERRO de leitura na serial: %v — reabrindo porta", err)
				porta.Close()
				time.Sleep(2 * time.Second)
				break
			}
			if n == 0 {
				continue
			}
			for _, l := range parser.Alimentar(buf[:n]) {
				leituras <- l
			}
		}
	}
}

// modoSniffer mostra os bytes crus recebidos — use para descobrir o formato
// que o TI-200 está enviando antes de configurar o protocolo.
func modoSniffer(cfg SerialConfig) {
	porta, err := abrirSerial(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer porta.Close()
	fmt.Printf("SNIFFER na porta %s (%d baud) — pendure um peso na balança. Ctrl+C para sair.\n\n", cfg.Porta, cfg.Baud)

	buf := make([]byte, 256)
	for {
		n, err := porta.Read(buf)
		if err != nil {
			log.Fatalf("erro de leitura: %v", err)
		}
		if n == 0 {
			continue
		}
		hexa := strings.Builder{}
		texto := strings.Builder{}
		for _, b := range buf[:n] {
			fmt.Fprintf(&hexa, "%02X ", b)
			if b >= 32 && b < 127 {
				texto.WriteByte(b)
			} else {
				texto.WriteByte('.')
			}
		}
		fmt.Printf("HEX: %-60s | TEXTO: %s\n", hexa.String(), texto.String())
	}
}
