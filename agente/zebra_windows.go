//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

// imprimirWindows envia bytes RAW (ZPL) direto para uma impressora instalada
// no Windows (USB/compartilhada), usando o spooler via winspool.drv.
func imprimirWindows(nomeImpressora, zpl string, copias int) error {
	if nomeImpressora == "" {
		return fmt.Errorf("configure o nome da impressora Windows em impressora.endereco")
	}

	winspool := syscall.NewLazyDLL("winspool.drv")
	openPrinter := winspool.NewProc("OpenPrinterW")
	closePrinter := winspool.NewProc("ClosePrinter")
	startDoc := winspool.NewProc("StartDocPrinterW")
	endDoc := winspool.NewProc("EndDocPrinter")
	startPage := winspool.NewProc("StartPagePrinter")
	endPage := winspool.NewProc("EndPagePrinter")
	writePrinter := winspool.NewProc("WritePrinter")

	nome, err := syscall.UTF16PtrFromString(nomeImpressora)
	if err != nil {
		return err
	}
	var h syscall.Handle
	r, _, errno := openPrinter.Call(uintptr(unsafe.Pointer(nome)), uintptr(unsafe.Pointer(&h)), 0)
	if r == 0 {
		return fmt.Errorf("impressora %q não encontrada: %v", nomeImpressora, errno)
	}
	defer closePrinter.Call(uintptr(h))

	docNome, _ := syscall.UTF16PtrFromString("Etiqueta de Pesagem")
	tipoRaw, _ := syscall.UTF16PtrFromString("RAW")
	docInfo := struct {
		pDocName    *uint16
		pOutputFile *uint16
		pDatatype   *uint16
	}{docNome, nil, tipoRaw}

	r, _, errno = startDoc.Call(uintptr(h), 1, uintptr(unsafe.Pointer(&docInfo)))
	if r == 0 {
		return fmt.Errorf("StartDocPrinter falhou: %v", errno)
	}
	defer endDoc.Call(uintptr(h))

	dados := []byte(zpl)
	for i := 0; i < copias; i++ {
		r, _, errno = startPage.Call(uintptr(h))
		if r == 0 {
			return fmt.Errorf("StartPagePrinter falhou: %v", errno)
		}
		var escritos uint32
		r, _, errno = writePrinter.Call(uintptr(h), uintptr(unsafe.Pointer(&dados[0])), uintptr(len(dados)), uintptr(unsafe.Pointer(&escritos)))
		endPage.Call(uintptr(h))
		if r == 0 || int(escritos) != len(dados) {
			return fmt.Errorf("WritePrinter falhou (%d/%d bytes): %v", escritos, len(dados), errno)
		}
	}
	return nil
}
