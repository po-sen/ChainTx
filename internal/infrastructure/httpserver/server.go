package httpserver

import (
	"context"
	stderrors "errors"
	"log"
	"net/http"
)

type Server struct {
	httpServer *http.Server
	logger     *log.Logger
}

func New(address string, handler http.Handler, logger *log.Logger) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:    address,
			Handler: handler,
		},
		logger: logger,
	}
}

func (s *Server) Start() error {
	s.logger.Printf("server starting address=%s", s.httpServer.Addr)

	err := s.httpServer.ListenAndServe()
	if err != nil && !stderrors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Printf("server shutting down")
	return s.httpServer.Shutdown(ctx)
}
