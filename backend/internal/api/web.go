package api

import (
	"embed"
	"net/http"
	"strings"
)

//go:embed web/index.html
var webApp embed.FS

func (s *Server) handleWebApp(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}
	page, err := webApp.ReadFile("web/index.html")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(page)
}
