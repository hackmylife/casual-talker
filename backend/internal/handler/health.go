package handler

import (
	"encoding/json"
	"net/http"
)

// HealthResponse represents the response body for the health check endpoint.
type HealthResponse struct {
	Status string `json:"status"`
}

// Health handles GET /api/v1/health and returns a simple status response.
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(HealthResponse{Status: "ok"})
}
