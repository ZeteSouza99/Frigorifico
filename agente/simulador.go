package main

import (
	"log"
	"math/rand"
	"time"
)

// simular gera leituras sintéticas imitando o ciclo real da balança de tendal:
// gancho vazio -> animal pendurado (peso sobe) -> estabiliza -> retirado.
// Use para testar o pipeline completo sem a balança conectada.
func simular(leituras chan<- Leitura, ciclos int) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	tick := 100 * time.Millisecond

	emitir := func(p float64) {
		leituras <- Leitura{PesoKg: p}
		time.Sleep(tick)
	}

	for c := 0; ciclos <= 0 || c < ciclos; c++ {
		alvo := 380 + rng.Float64()*180 // peso do animal

		log.Printf("[simulador] ciclo %d: pendurando animal de ~%.1f kg", c+1, alvo)
		for i := 0; i < 8; i++ { // gancho vazio
			emitir(rng.Float64() * 2)
		}
		for f := 0.2; f < 1.0; f += 0.2 { // subindo (instável)
			emitir(alvo*f + rng.Float64()*20 - 10)
		}
		for i := 0; i < 15; i++ { // estável
			emitir(alvo + rng.Float64()*0.4 - 0.2)
		}
		for f := 0.8; f > 0; f -= 0.25 { // retirando
			emitir(alvo * f)
		}
		for i := 0; i < 5; i++ {
			emitir(rng.Float64() * 2)
		}
	}
	close(leituras)
}
