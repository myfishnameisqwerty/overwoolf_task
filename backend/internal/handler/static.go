package handler

import (
	"net/http"
	"os"
)

func (h *Handler) ServeWidgetJS(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile(h.WidgetPath)
	if err != nil {
		writeError(w, http.StatusNotFound, "widget not found")
		return
	}
	w.Header().Set("Content-Type", "application/javascript")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (h *Handler) ServeDashboard(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile(h.DashboardPath)
	if err != nil {
		writeError(w, http.StatusNotFound, "dashboard not found")
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

