package dados

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// PGStore implementa Store usando PostgreSQL (Neon).
type PGStore struct {
	db  *sql.DB
	loc *time.Location
	tz  string
}

const schema = `
CREATE TABLE IF NOT EXISTS pesagens (
    id        BIGSERIAL PRIMARY KEY,
    brinco    TEXT NOT NULL DEFAULT '',
    peso_kg   NUMERIC(7,2) NOT NULL CHECK (peso_kg > 0),
    data_hora TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_pesagens_data_hora ON pesagens (data_hora);
`

// NewPGStore conecta no Neon, valida a conexão e garante o schema.
func NewPGStore(dsn string, loc *time.Location) (*PGStore, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping falhou: %w", err)
	}
	if _, err := db.ExecContext(ctx, schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("erro ao criar schema: %w", err)
	}
	return &PGStore{db: db, loc: loc, tz: loc.String()}, nil
}

func (s *PGStore) AddPesagem(ctx context.Context, p Pesagem) (Pesagem, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO pesagens (brinco, peso_kg, data_hora) VALUES ($1, $2, $3) RETURNING id`,
		p.Brinco, p.PesoKg, p.DataHora,
	).Scan(&p.ID)
	return p, err
}

func (s *PGStore) ListPesagens(ctx context.Context, dia time.Time) ([]Pesagem, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, brinco, peso_kg::float8, data_hora
		FROM pesagens
		WHERE (data_hora AT TIME ZONE $2)::date = $1::date
		ORDER BY data_hora DESC`,
		dia.Format("2006-01-02"), s.tz)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Pesagem{}
	for rows.Next() {
		var p Pesagem
		if err := rows.Scan(&p.ID, &p.Brinco, &p.PesoKg, &p.DataHora); err != nil {
			return nil, err
		}
		p.DataHora = p.DataHora.In(s.loc)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *PGStore) Resumo(ctx context.Context, dia time.Time) (Resumo, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	res := Resumo{Data: dia.Format("2006-01-02"), MediaPorHorario: []MediaHorario{}}
	diaStr := dia.Format("2006-01-02")

	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(AVG(peso_kg), 0)::float8,
		       COUNT(*),
		       COALESCE(MAX(peso_kg), 0)::float8,
		       COALESCE(MIN(peso_kg), 0)::float8
		FROM pesagens
		WHERE (data_hora AT TIME ZONE $2)::date = $1::date`,
		diaStr, s.tz,
	).Scan(&res.MediaPesoKg, &res.TotalPesagens, &res.MaiorPesoKg, &res.MenorPesoKg)
	if err != nil {
		return res, err
	}
	res.MediaPesoKg = round2(res.MediaPesoKg)

	rows, err := s.db.QueryContext(ctx, `
		SELECT EXTRACT(HOUR FROM data_hora AT TIME ZONE $2)::int AS hora,
		       AVG(peso_kg)::float8,
		       COUNT(*)
		FROM pesagens
		WHERE (data_hora AT TIME ZONE $2)::date = $1::date
		GROUP BY hora
		ORDER BY hora`,
		diaStr, s.tz)
	if err != nil {
		return res, err
	}
	defer rows.Close()

	for rows.Next() {
		var m MediaHorario
		if err := rows.Scan(&m.Hora, &m.MediaKg, &m.Total); err != nil {
			return res, err
		}
		m.MediaKg = round2(m.MediaKg)
		res.MediaPorHorario = append(res.MediaPorHorario, m)
	}
	return res, rows.Err()
}

func (s *PGStore) Close() error { return s.db.Close() }
