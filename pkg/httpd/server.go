package httpd

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

var ShutdownTimeout = time.Second * 5

type Server struct {
	addr   string
	lnAddr string
	logger *zap.Logger
	srv    *http.Server
	done   sync.WaitGroup
	err    error
}

func New(addr string, h http.Handler) *Server {
	return &Server{addr: addr, srv: &http.Server{Handler: h}, logger: zap.NewNop()}
}

func (s *Server) SetLogger(l *zap.Logger) {
	s.logger = l
}

func (s *Server) Addr() string {
	return s.lnAddr
}

func (s *Server) Start(ctx context.Context) error {
	s.done.Add(1)
	s.srv.BaseContext = func(l net.Listener) context.Context { return ctx }
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		s.logger.Error("Listen error", zap.Error(err))
		return err
	}
	s.lnAddr = ln.Addr().String()
	s.logger.Info("Listening", zap.String("addr", s.lnAddr))
	go s.serve(ctx, ln)
	return nil
}

func (s *Server) Run(ctx context.Context) error {
	if err := s.Start(ctx); err != nil {
		return err
	}
	return s.Wait()
}

func (s *Server) Wait() error {
	s.done.Wait()
	return s.err
}

func (s *Server) serve(ctx context.Context, ln net.Listener) {
	defer s.done.Done()
	errCh := make(chan error)
	go func() {
		if err := s.srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()
	// Wait for either the context to cancel or for the server to exit.
	var reason string
	var srvErr error
	select {
	case srvErr = <-errCh:
		reason = "server error"
	case <-ctx.Done():
		reason = "context closed"
	}

	s.logger.Info("Shutting down", zap.String("reason", reason), zap.Error(srvErr))
	timeout, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
	defer cancel()
	err := s.srv.Shutdown(timeout)
	if errors.Is(err, http.ErrServerClosed) {
		err = srvErr
	} else if errors.Is(err, context.DeadlineExceeded) {
		s.logger.Warn("Server shutdown deadline exceeded", zap.Duration("duration", ShutdownTimeout))
		if clsErr := s.srv.Close(); clsErr != nil {
			err = clsErr
		}
	}
	s.logger.Info("Closed", zap.Error(err))
	s.err = err
}
