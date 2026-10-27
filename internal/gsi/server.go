package gsi

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

const maxBodyBytes = 1 << 20

type Server struct {
	token string
	log   *zap.Logger

	mu     sync.Mutex
	latest State
	at     time.Time
}

func NewServer(token string, log *zap.Logger) *Server {
	return &Server{token: token, log: log}
}

func (s *Server) ListenAndServe(addr string) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           http.HandlerFunc(s.handle),
		ReadHeaderTimeout: 5 * time.Second,
	}

	return srv.ListenAndServe()
}

func (s *Server) Latest() (State, time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.latest, s.at
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var state State

	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&state); err != nil {
		s.log.Warn("gsi: decode", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if subtle.ConstantTimeCompare([]byte(state.Auth.Token), []byte(s.token)) != 1 {
		s.log.Warn("gsi: invalid token")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	s.mu.Lock()
	s.latest = state
	s.at = time.Now()
	s.mu.Unlock()

	w.WriteHeader(http.StatusOK)
}
