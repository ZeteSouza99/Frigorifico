package dados

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"
)

// MemStore guarda pesagens em memória — usado quando DATABASE_URL não está definido.
type MemStore struct {
	mu    sync.RWMutex
	loc   *time.Location
	seq   int64
	pesos []Pesagem
}

func NewMemStore(loc *time.Location, seed bool) *MemStore {
	s := &MemStore{loc: loc}
	if seed {
		s.seedDemo()
	}
	return s
}

func (s *MemStore) seedDemo() {
	rng := rand.New(rand.NewSource(42))
	now := time.Now().In(s.loc)
	dia := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, s.loc)
	brincos := []string{"BR", "NL", "GO", "MT"}
	for h := 7; h <= 17; h++ {
		n := 2 + rng.Intn(5)
		for i := 0; i < n; i++ {
			ts := dia.Add(time.Duration(h)*time.Hour + time.Duration(rng.Intn(60))*time.Minute)
			if ts.After(now) {
				continue
			}
			s.seq++
			s.pesos = append(s.pesos, Pesagem{
				ID:       s.seq,
				Brinco:   fmt.Sprintf("%s-%s-%03d", brincos[rng.Intn(len(brincos))], now.Format("06"), s.seq),
				PesoKg:   round2(360 + rng.Float64()*220),
				DataHora: ts,
			})
		}
	}
}

func (s *MemStore) AddPesagem(_ context.Context, p Pesagem) (Pesagem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	p.ID = s.seq
	p.PesoKg = round2(p.PesoKg)
	s.pesos = append(s.pesos, p)
	return p, nil
}

func (s *MemStore) doDia(dia time.Time) []Pesagem {
	var out []Pesagem
	for _, p := range s.pesos {
		t := p.DataHora.In(s.loc)
		if t.Year() == dia.Year() && t.YearDay() == dia.YearDay() {
			out = append(out, p)
		}
	}
	return out
}

func (s *MemStore) ListPesagens(_ context.Context, dia time.Time) ([]Pesagem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := s.doDia(dia)
	sort.Slice(out, func(i, j int) bool { return out[i].DataHora.After(out[j].DataHora) })
	if out == nil {
		out = []Pesagem{}
	}
	return out, nil
}

func (s *MemStore) Resumo(_ context.Context, dia time.Time) (Resumo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	res := Resumo{Data: dia.Format("2006-01-02"), MediaPorHorario: []MediaHorario{}}
	itens := s.doDia(dia)
	if len(itens) == 0 {
		return res, nil
	}

	var soma float64
	maior, menor := itens[0].PesoKg, itens[0].PesoKg
	somaHora := map[int]float64{}
	contHora := map[int]int{}
	for _, p := range itens {
		soma += p.PesoKg
		if p.PesoKg > maior {
			maior = p.PesoKg
		}
		if p.PesoKg < menor {
			menor = p.PesoKg
		}
		h := p.DataHora.In(s.loc).Hour()
		somaHora[h] += p.PesoKg
		contHora[h]++
	}

	res.TotalPesagens = len(itens)
	res.MediaPesoKg = round2(soma / float64(len(itens)))
	res.MaiorPesoKg = maior
	res.MenorPesoKg = menor
	for h := 0; h < 24; h++ {
		if contHora[h] > 0 {
			res.MediaPorHorario = append(res.MediaPorHorario, MediaHorario{
				Hora:    h,
				MediaKg: round2(somaHora[h] / float64(contHora[h])),
				Total:   contHora[h],
			})
		}
	}
	return res, nil
}

func (s *MemStore) Close() error { return nil }
