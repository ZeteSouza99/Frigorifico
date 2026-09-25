//go:build !windows

package main

import "fmt"

func imprimirWindows(nomeImpressora, zpl string, copias int) error {
	return fmt.Errorf("impressora tipo 'windows' só está disponível no Windows; use 'rede'")
}
