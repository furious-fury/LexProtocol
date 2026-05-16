package api

import (
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"strings"

	"github.com/lexprotocol/lexprotocol/backend/internal/pricing"
)

type Server struct {
	pricing *pricing.Service
}

func NewServer(pricingService *pricing.Service) http.Handler {
	server := &Server{pricing: pricingService}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", server.healthz)
	mux.HandleFunc("GET /price/{marketId}", server.price)
	mux.HandleFunc("GET /signed/{marketId}", server.signed)
	return mux
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) price(w http.ResponseWriter, r *http.Request) {
	marketID, ok := parseMarketID(w, r)
	if !ok {
		return
	}

	quote, err := s.pricing.Price(r.Context(), marketID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, quote)
}

func (s *Server) signed(w http.ResponseWriter, r *http.Request) {
	marketID, ok := parseMarketID(w, r)
	if !ok {
		return
	}

	outcomeID, err := pricing.OutcomeID(r.URL.Query().Get("outcome"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	submission, err := s.pricing.SignedSettlement(r.Context(), marketID, outcomeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, submission)
}

func parseMarketID(w http.ResponseWriter, r *http.Request) (*big.Int, bool) {
	raw := strings.TrimSpace(r.PathValue("marketId"))
	marketID, ok := new(big.Int).SetString(raw, 10)
	if !ok || marketID.Sign() <= 0 {
		writeError(w, http.StatusBadRequest, errors.New("marketId must be a positive integer"))
		return nil, false
	}
	return marketID, true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
