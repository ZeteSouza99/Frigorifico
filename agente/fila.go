package main

import (
	"bufio"
	"encoding/json"
	"os"
	"sync"
)

// Fila guarda pesagens que falharam no envio (sem internet) em um arquivo
// JSON Lines e as reenvia quando a conexão voltar.
type Fila struct {
	mu      sync.Mutex
	caminho string
}

func NovaFila(caminho string) *Fila {
	return &Fila{caminho: caminho}
}

func (f *Fila) Adicionar(p PesagemAPI) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	arq, err := os.OpenFile(f.caminho, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer arq.Close()
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = arq.Write(append(b, '\n'))
	return err
}

// Reenviar tenta enviar todas as pendências; devolve (enviadas, restantes).
func (f *Fila) Reenviar(envia func(PesagemAPI) error) (int, int) {
	f.mu.Lock()
	defer f.mu.Unlock()

	arq, err := os.Open(f.caminho)
	if os.IsNotExist(err) {
		return 0, 0
	}
	if err != nil {
		return 0, -1
	}
	var pendentes []PesagemAPI
	sc := bufio.NewScanner(arq)
	for sc.Scan() {
		var p PesagemAPI
		if json.Unmarshal(sc.Bytes(), &p) == nil {
			pendentes = append(pendentes, p)
		}
	}
	arq.Close()

	if len(pendentes) == 0 {
		os.Remove(f.caminho)
		return 0, 0
	}

	var restantes []PesagemAPI
	enviadas := 0
	for i, p := range pendentes {
		if err := envia(p); err != nil {
			// falhou: mantém esta e todas as seguintes para preservar a ordem
			restantes = append(restantes, pendentes[i:]...)
			break
		}
		enviadas++
	}

	if len(restantes) == 0 {
		os.Remove(f.caminho)
		return enviadas, 0
	}
	tmp := f.caminho + ".tmp"
	saida, err := os.Create(tmp)
	if err != nil {
		return enviadas, len(restantes)
	}
	for _, p := range restantes {
		b, _ := json.Marshal(p)
		saida.Write(append(b, '\n'))
	}
	saida.Close()
	os.Rename(tmp, f.caminho)
	return enviadas, len(restantes)
}
