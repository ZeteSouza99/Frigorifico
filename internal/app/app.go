// Package app contém os handlers HTTP compartilhados entre o servidor local
// e as serverless functions do Vercel. A inicialização é preguiçosa (lazy)
// para reaproveitar a conexão entre invocações da mesma instância.
package app

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	_ "time/tzdata"

	"frigorifico-dashboard/internal/dados"
)

var (
	initOnce sync.Once
	store    dados.Store
	loc      *time.Location
	initErr  error
)

func ensureInit() error {
	initOnce.Do(func() {
		tz := envOr("APP_TZ", "America/Manaus")
		loc, initErr = time.LoadLocation(tz)
		if initErr != nil {
			initErr = errors.New("fuso horário inválido em APP_TZ: " + tz)
			return
		}
		if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
			store, initErr = dados.NewPGStore(dsn, loc)
			if initErr == nil {
				log.Println("conectado ao banco Neon/PostgreSQL")
			}
		} else {
			store = dados.NewMemStore(loc, true)
			log.Println("AVISO: DATABASE_URL não definido — rodando em MODO DEMONSTRAÇÃO (dados em memória)")
		}
	})
	return initErr
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// parseDia interpreta ?data=YYYY-MM-DD (padrão: hoje no fuso configurado).
func parseDia(r *http.Request) (time.Time, error) {
	s := strings.TrimSpace(r.URL.Query().Get("data"))
	if s == "" {
		now := time.Now().In(loc)
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc), nil
	}
	d, err := time.ParseInLocation("2006-01-02", s, loc)
	if err != nil {
		return time.Time{}, errors.New("parâmetro 'data' inválido, use o formato YYYY-MM-DD")
	}
	return d, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"erro": msg})
}

// Resumo trata GET /api/resumo.
func Resumo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "método não permitido")
		return
	}
	if err := ensureInit(); err != nil {
		log.Printf("erro de inicialização: %v", err)
		writeErr(w, http.StatusInternalServerError, "erro de configuração do servidor")
		return
	}
	dia, err := parseDia(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	res, err := store.Resumo(r.Context(), dia)
	if err != nil {
		log.Printf("erro no resumo: %v", err)
		writeErr(w, http.StatusInternalServerError, "erro ao consultar o resumo")
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// Pesagens trata GET (lista do dia) e POST (registro) em /api/pesagens.
func Pesagens(w http.ResponseWriter, r *http.Request) {
	if err := ensureInit(); err != nil {
		log.Printf("erro de inicialização: %v", err)
		writeErr(w, http.StatusInternalServerError, "erro de configuração do servidor")
		return
	}
	switch r.Method {
	case http.MethodGet:
		listPesagens(w, r)
	case http.MethodPost:
		addPesagem(w, r)
	default:
		writeErr(w, http.StatusMethodNotAllowed, "método não permitido")
	}
}

func listPesagens(w http.ResponseWriter, r *http.Request) {
	dia, err := parseDia(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	itens, err := store.ListPesagens(r.Context(), dia)
	if err != nil {
		log.Printf("erro ao listar pesagens: %v", err)
		writeErr(w, http.StatusInternalServerError, "erro ao listar pesagens")
		return
	}
	writeJSON(w, http.StatusOK, itens)
}

type novaPesagem struct {
	Brinco   string  `json:"brinco"`
	PesoKg   float64 `json:"peso_kg"`
	DataHora string  `json:"data_hora"` // RFC3339 opcional; padrão: agora
}

func addPesagem(w http.ResponseWriter, r *http.Request) {
	if key := os.Getenv("API_KEY"); key != "" && r.Header.Get("X-API-Key") != key {
		writeErr(w, http.StatusUnauthorized, "chave de API inválida ou ausente")
		return
	}
	var in novaPesagem
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if in.PesoKg <= 0 || in.PesoKg > 2000 {
		writeErr(w, http.StatusBadRequest, "peso_kg deve estar entre 0 e 2000")
		return
	}
	quando := time.Now().In(loc)
	if in.DataHora != "" {
		t, err := time.Parse(time.RFC3339, in.DataHora)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "data_hora inválida, use o formato RFC3339")
			return
		}
		quando = t.In(loc)
	}
	p, err := store.AddPesagem(r.Context(), dados.Pesagem{
		Brinco:   strings.TrimSpace(in.Brinco),
		PesoKg:   in.PesoKg,
		DataHora: quando,
	})
	if err != nil {
		log.Printf("erro ao registrar pesagem: %v", err)
		writeErr(w, http.StatusInternalServerError, "erro ao registrar pesagem")
		return
	}
	writeJSON(w, http.StatusCreated, p)
}
