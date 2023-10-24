package httpserver

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type handler interface {
	Routes(r chi.Router)
}

type Server struct {
	serviceName string
	address     string
	timeout     time.Duration
	fileHandler handler
	logger      *zap.Logger
}

// NewServer returns a new HTTP server with all dependencies setup.
func NewServer(serviceName string, address string, timeout time.Duration, fileHandler handler, logger *zap.Logger) Server {
	return Server{
		serviceName: serviceName,
		address:     address,
		timeout:     timeout,
		fileHandler: fileHandler,
		logger:      logger,
	}
}

// Open setups a TCP listener on the servers address and serves http requests.
func (s Server) Open() error {
	ln, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}

	server := http.Server{
		Handler: http.TimeoutHandler(s.handler(), s.timeout, "http request timed out"),
	}

	s.logger.Info(fmt.Sprintf("http server listening on %s", s.address))

	return server.Serve(ln)
}

func (s Server) handler() http.Handler {
	router := chi.NewRouter()

	// Logs incoming requests.
	router.Use(chimiddleware.Logger)

	// Recovers and logs panics.
	router.Use(chimiddleware.Recoverer)

	// Health endpoint.
	router.Use(chimiddleware.Heartbeat("/health"))

	// Prometheus metrics.
	router.Handle("/metrics", promhttp.Handler())

	// v1 handlers.
	router.Route("/api/v1", func(r chi.Router) {
		r.Route("/file", s.fileHandler.Routes)
	})

	return router
}
