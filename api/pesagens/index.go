// Serverless function do Vercel: GET e POST /api/pesagens
package handler

import (
	"net/http"

	"frigorifico-dashboard/internal/app"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	app.Pesagens(w, r)
}
