package httpserver

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/xxyGoTop/ai-stock/services/api-go/internal/indicator"
	"github.com/xxyGoTop/ai-stock/services/api-go/internal/provider"
	"github.com/xxyGoTop/ai-stock/services/api-go/pkg/response"
)

type Server struct {
	bundle *provider.Bundle
	origin string
}

func New(timeout time.Duration) *Server {
	origin := os.Getenv("WEB_ORIGIN")
	if origin == "" {
		origin = "http://localhost:5173"
	}
	return &Server{bundle: provider.NewProviders(timeout), origin: origin}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.health)
	mux.HandleFunc("/api/v1/stocks/search", s.search)
	mux.HandleFunc("/api/v1/stocks/", s.stock)
	mux.HandleFunc("/api/v1/quotes/indices", s.indices)
	mux.HandleFunc("/api/v1/quotes", s.quote)
	mux.HandleFunc("/api/v1/kline", s.kline)
	mux.HandleFunc("/api/v1/indicators", s.indicators)
	return s.cors(mux)
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", s.origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	response.OK(w, map[string]string{"service": "api-go"})
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		response.Error(w, http.StatusBadRequest, "q required")
		return
	}
	items, err := s.bundle.Search(q)
	if err != nil {
		response.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(w, items)
}

func (s *Server) stock(w http.ResponseWriter, r *http.Request) {
	symbol := strings.TrimPrefix(r.URL.Path, "/api/v1/stocks/")
	s.writeQuote(w, symbol)
}

func (s *Server) quote(w http.ResponseWriter, r *http.Request) {
	s.writeQuote(w, r.URL.Query().Get("symbol"))
}

func (s *Server) writeQuote(w http.ResponseWriter, symbol string) {
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		response.Error(w, http.StatusBadRequest, "symbol required")
		return
	}
	q, err := s.bundle.Quote(symbol)
	if err != nil {
		response.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(w, q)
}

func (s *Server) indices(w http.ResponseWriter, _ *http.Request) {
	items, err := s.bundle.IndexQuotes()
	if err != nil {
		response.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(w, items)
}

func (s *Server) kline(w http.ResponseWriter, r *http.Request) {
	res, err := s.loadKline(r)
	if err != nil {
		status := http.StatusBadGateway
		if err == errSymbol {
			status = http.StatusBadRequest
		}
		response.Error(w, status, err.Error())
		return
	}
	response.OK(w, res)
}

func (s *Server) indicators(w http.ResponseWriter, r *http.Request) {
	symbol := strings.TrimSpace(r.URL.Query().Get("symbol"))
	if symbol == "" {
		response.Error(w, http.StatusBadRequest, errSymbol.Error())
		return
	}
	res, err := s.bundle.AggregatedKline(symbol, 180)
	if err != nil {
		response.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	series := indicator.Compute(res.Bars)
	var latest *indicator.Point
	if len(series) > 0 {
		p := series[len(series)-1]
		latest = &p
	}
	response.OK(w, indicator.Result{
		Symbol:    provider.PadSymbol(r.URL.Query().Get("symbol")),
		Freshness: res.Freshness,
		Latest:    latest,
		Series:    series,
	})
}

func (s *Server) loadKline(r *http.Request) (*provider.KlineResult, error) {
	symbol := strings.TrimSpace(r.URL.Query().Get("symbol"))
	if symbol == "" {
		return nil, errSymbol
	}
	limit := 180
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 800 {
			limit = n
		}
	}
	return s.bundle.AggregatedKline(symbol, limit)
}

var errSymbol = simpleError("symbol required")

type simpleError string

func (e simpleError) Error() string { return string(e) }
