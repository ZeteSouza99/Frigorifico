// Package dados define os tipos e o acesso a dados das pesagens.
package dados

import (
	"context"
	"math"
	"time"
)

// Pesagem representa uma pesagem individual de um animal.
type Pesagem struct {
	ID       int64     `json:"id"`
	Brinco   string    `json:"brinco,omitempty"`
	PesoKg   float64   `json:"peso_kg"`
	DataHora time.Time `json:"data_hora"`
}

// MediaHorario é a média de peso agrupada por hora do dia.
type MediaHorario struct {
	Hora    int     `json:"hora"`
	MediaKg float64 `json:"media_kg"`
	Total   int     `json:"total"`
}

// Resumo agrega os indicadores do dashboard para um dia.
type Resumo struct {
	Data            string         `json:"data"`
	MediaPesoKg     float64        `json:"media_peso_kg"`
	TotalPesagens   int            `json:"total_pesagens"`
	MaiorPesoKg     float64        `json:"maior_peso_kg"`
	MenorPesoKg     float64        `json:"menor_peso_kg"`
	MediaPorHorario []MediaHorario `json:"media_por_horario"`
}

// Store abstrai a persistência (PostgreSQL/Neon ou memória em modo demo).
type Store interface {
	AddPesagem(ctx context.Context, p Pesagem) (Pesagem, error)
	ListPesagens(ctx context.Context, dia time.Time) ([]Pesagem, error)
	Resumo(ctx context.Context, dia time.Time) (Resumo, error)
	Close() error
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }
