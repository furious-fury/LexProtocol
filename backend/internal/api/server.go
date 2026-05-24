package api

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"strings"

	"github.com/lexprotocol/lexprotocol/backend/internal/pricing"
)

type Server struct {
	pricing  *pricing.Service
	apiToken string
}

type Option func(*Server)

func WithSignedAPIToken(token string) Option {
	return func(server *Server) {
		server.apiToken = strings.TrimSpace(token)
	}
}

func NewServer(pricingService *pricing.Service, options ...Option) http.Handler {
	server := &Server{pricing: pricingService}
	for _, option := range options {
		option(server)
	}
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
	if !s.authorized(r) {
		writeError(w, http.StatusUnauthorized, errors.New("missing or invalid authorization token"))
		return
	}

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

func (s *Server) authorized(r *http.Request) bool {
	if s.apiToken == "" {
		return true
	}
	raw := strings.TrimSpace(r.Header.Get("Authorization"))
	const prefix = "Bearer "
	if !strings.HasPrefix(raw, prefix) {
		return false
	}
	token := strings.TrimSpace(strings.TrimPrefix(raw, prefix))
	return subtle.ConstantTimeCompare([]byte(token), []byte(s.apiToken)) == 1
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
