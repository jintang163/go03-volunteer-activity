package server

import (
	"net/http"

	"go03-volunteer-activity/internal/handler"
)

func NewMux(h *handler.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	h.Routes(mux)
	return mux
}
