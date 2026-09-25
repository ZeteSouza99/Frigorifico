package main

import "math"

// Detector identifica peso estável: N leituras consecutivas dentro da tolerância,
// acima do peso mínimo. Após capturar, exige que o peso caia abaixo do rearme
// antes de permitir nova captura (evita registrar o mesmo animal duas vezes).
type Detector struct {
	cfg     CapturaConfig
	ultimas []float64
	armado  bool
}

func NovoDetector(cfg CapturaConfig) *Detector {
	if cfg.LeiturasEstaveis < 2 {
		cfg.LeiturasEstaveis = 2
	}
	return &Detector{cfg: cfg, armado: true}
}

// Alimentar processa uma leitura e retorna (peso, true) quando uma pesagem
// estável for detectada.
func (d *Detector) Alimentar(l Leitura) (float64, bool) {
	p := l.PesoKg

	if !d.armado {
		if p <= d.cfg.RearmeKg {
			d.armado = true
			d.ultimas = nil
		}
		return 0, false
	}

	if l.MovimentoValido && l.EmMovimento {
		d.ultimas = nil
		return 0, false
	}

	d.ultimas = append(d.ultimas, p)
	if len(d.ultimas) > d.cfg.LeiturasEstaveis {
		d.ultimas = d.ultimas[1:]
	}
	if len(d.ultimas) < d.cfg.LeiturasEstaveis {
		return 0, false
	}

	menor, maior, soma := d.ultimas[0], d.ultimas[0], 0.0
	for _, v := range d.ultimas {
		if v < menor {
			menor = v
		}
		if v > maior {
			maior = v
		}
		soma += v
	}
	if maior-menor > d.cfg.ToleranciaKg {
		return 0, false
	}
	media := soma / float64(len(d.ultimas))
	if media < d.cfg.PesoMinimoKg {
		return 0, false
	}

	d.armado = false
	d.ultimas = nil
	return math.Round(media*100) / 100, true
}
