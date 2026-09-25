// Agente de balança — Toledo Tendal 200 (indicador TI-200).
//
// Lê o peso pela porta serial RS-232, detecta pesagem estável, envia para a
// API do dashboard (local ou Vercel) e imprime etiqueta ZPL na Zebra.
// Pesagens feitas sem internet ficam em fila e são reenviadas depois.
//
// Uso:
//
//	agente-balanca.exe            inicia o agente (cria agente.json na 1ª vez)
//	agente-balanca.exe -listar    lista as portas seriais disponíveis
//	agente-balanca.exe -sniffer   mostra os bytes crus da balança (descobrir protocolo)
//	agente-balanca.exe -simular   testa o pipeline completo sem a balança
//	agente-balanca.exe -testar-impressao   imprime uma etiqueta de teste
package main

import (
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

func main() {
	// trabalha sempre na pasta do executável (config, log, fila e template)
	if exe, err := os.Executable(); err == nil {
		os.Chdir(filepath.Dir(exe))
	}

	log.SetFlags(log.Ldate | log.Ltime)
	if arqLog, err := os.OpenFile("agente.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
		log.SetOutput(io.MultiWriter(os.Stdout, arqLog))
		defer arqLog.Close()
	}

	caminhoCfg := flag.String("config", "agente.json", "caminho do arquivo de configuração")
	fListar := flag.Bool("listar", false, "lista as portas seriais e sai")
	fSniffer := flag.Bool("sniffer", false, "mostra os bytes crus da serial e sai")
	fSimular := flag.Bool("simular", false, "gera pesagens sintéticas (teste sem balança)")
	fCiclos := flag.Int("ciclos", 0, "com -simular: quantidade de animais a simular (0 = infinito)")
	fTestarImp := flag.Bool("testar-impressao", false, "imprime uma etiqueta de teste e sai")
	flag.Parse()

	if *fListar {
		listarPortas()
		return
	}

	cfg, criado, err := carregarConfig(*caminhoCfg)
	if err != nil {
		log.Fatal(err)
	}
	if criado {
		log.Printf("arquivo %s criado com os valores padrão — revise a porta serial e a URL da API", *caminhoCfg)
	}

	template, err := carregarTemplate(cfg.Impressora.Template)
	if err != nil {
		log.Fatalf("erro no template da etiqueta: %v", err)
	}

	if *fTestarImp {
		d := DadosEtiqueta{PesoKg: 487.5, Brinco: "TESTE", Codigo: time.Now().Format("20060102-150405"), Quando: time.Now()}
		if err := Imprimir(cfg.Impressora, renderizarEtiqueta(template, d), d.Codigo); err != nil {
			log.Fatalf("ERRO na impressão de teste: %v", err)
		}
		log.Printf("etiqueta de teste enviada (tipo=%s, endereco=%s)", cfg.Impressora.Tipo, cfg.Impressora.Endereco)
		return
	}

	if *fSniffer {
		modoSniffer(cfg.Serial)
		return
	}

	cliente := NovoClienteAPI(cfg.API)
	fila := NovaFila("pendentes.jsonl")
	detector := NovoDetector(cfg.Captura)
	leituras := make(chan Leitura, 128)

	// reenvio das pesagens pendentes: na inicialização e a cada 30s
	go func() {
		for {
			if enviadas, restantes := fila.Reenviar(cliente.Enviar); enviadas > 0 || restantes > 0 {
				log.Printf("fila offline: %d reenviada(s), %d pendente(s)", enviadas, restantes)
			}
			time.Sleep(30 * time.Second)
		}
	}()

	if *fSimular {
		log.Println("MODO SIMULAÇÃO — gerando pesagens sintéticas")
		go simular(leituras, *fCiclos)
	} else {
		go lerSerial(cfg.Serial, leituras)
	}

	log.Printf("agente iniciado — API: %s | impressora: %s (%s)", cfg.API.URL, cfg.Impressora.Tipo, cfg.Impressora.Endereco)
	log.Printf("aguardando pesagens (mínimo %.0f kg, %d leituras estáveis, tolerância %.1f kg)",
		cfg.Captura.PesoMinimoKg, cfg.Captura.LeiturasEstaveis, cfg.Captura.ToleranciaKg)

	for l := range leituras {
		peso, capturou := detector.Alimentar(l)
		if !capturou {
			continue
		}
		registrar(cfg, cliente, fila, template, peso)
	}
}

func registrar(cfg Config, cliente *ClienteAPI, fila *Fila, template string, peso float64) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("ERRO inesperado ao registrar pesagem: %v", r)
		}
	}()
	agora := time.Now()
	codigo := agora.Format("20060102-150405")
	log.Printf(">>> PESAGEM ESTÁVEL: %.1f kg (código %s)", peso, codigo)

	p := PesagemAPI{PesoKg: peso, DataHora: agora.Format(time.RFC3339)}
	if err := cliente.Enviar(p); err != nil {
		log.Printf("    sem conexão com a API (%v) — pesagem guardada na fila offline", err)
		if err := fila.Adicionar(p); err != nil {
			log.Printf("    ERRO GRAVE ao salvar na fila: %v", err)
		}
	} else {
		log.Printf("    enviada para a API com sucesso")
	}

	d := DadosEtiqueta{PesoKg: peso, Codigo: codigo, Quando: agora}
	if err := Imprimir(cfg.Impressora, renderizarEtiqueta(template, d), codigo); err != nil {
		log.Printf("    ERRO na impressão da etiqueta: %v", err)
	} else if cfg.Impressora.Tipo != "desativada" {
		log.Printf("    etiqueta impressa (%s)", cfg.Impressora.Tipo)
	}
}
