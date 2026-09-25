// Serverless function do Vercel: GET /api/resumo
package handler

import (
	"net/http"

	"frigorifico-dashboard/internal/app"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	app.Resumo(w, r)
}
