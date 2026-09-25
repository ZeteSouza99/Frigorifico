package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// PesagemAPI é o corpo enviado para POST /api/pesagens.
type PesagemAPI struct {
	PesoKg   float64 `json:"peso_kg"`
	Brinco   string  `json:"brinco,omitempty"`
	DataHora string  `json:"data_hora"` // RFC3339
}

type ClienteAPI struct {
	cfg  APIConfig
	http *http.Client
}

func NovoClienteAPI(cfg APIConfig) *ClienteAPI {
	return &ClienteAPI{cfg: cfg, http: &http.Client{Timeout: 10 * time.Second}}
}

func (c *ClienteAPI) Enviar(p PesagemAPI) error {
	corpo, err := json.Marshal(p)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, c.cfg.URL+"/api/pesagens", bytes.NewReader(corpo))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.Chave != "" {
		req.Header.Set("X-API-Key", c.cfg.Chave)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("API respondeu %d: %s", resp.StatusCode, string(b))
	}
	return nil
}
