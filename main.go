// Servidor local: serve o frontend embutido e a mesma API usada no Vercel.
package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"

	"frigorifico-dashboard/internal/app"
)

//go:embed public
var webFS embed.FS

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/resumo", app.Resumo)
	mux.HandleFunc("/api/pesagens", app.Pesagens)

	webRoot, err := fs.Sub(webFS, "public")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("/", http.FileServer(http.FS(webRoot)))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("dashboard disponível em http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
