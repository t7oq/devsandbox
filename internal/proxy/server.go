package proxy

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"syscall"

	"github.com/elazarl/goproxy"
)

type Server struct {
	config    *Config
	ca        *CA
	proxy     *goproxy.ProxyHttpServer
	listener  net.Listener
	server    *http.Server
	logger    *log.Logger
	reqLogger *RequestLogger
	wg        sync.WaitGroup
	mu        sync.Mutex
	running   bool
}

func NewServer(cfg *Config) (*Server, error) {
	ca, err := LoadOrCreateCA(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to load/create CA: %w", err)
	}

	proxy := goproxy.NewProxyHttpServer()

	var logger *log.Logger
	if cfg.LogEnabled {
		logger = log.New(os.Stderr, "[proxy] ", log.LstdFlags)
		proxy.Verbose = true
	}

	// Create request logger for persisting full request/response data
	reqLogger, err := NewRequestLogger(cfg.LogDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create request logger: %w", err)
	}

	s := &Server{
		config:    cfg,
		ca:        ca,
		proxy:     proxy,
		logger:    logger,
		reqLogger: reqLogger,
	}

	s.setupMITM()
	s.setupLogging()

	return s, nil
}

func (s *Server) setupMITM() {
	// Configure MITM for all HTTPS connections
	s.proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)

	// Set up certificate generation
	goproxy.GoproxyCa = tls.Certificate{
		Certificate: [][]byte{s.ca.Certificate.Raw},
		PrivateKey:  s.ca.PrivateKey,
		Leaf:        s.ca.Certificate,
	}

	// Use our CA for signing
	tlsConfig := goproxy.TLSConfigFromCA(&goproxy.GoproxyCa)
	goproxy.MitmConnect.TLSConfig = func(host string, ctx *goproxy.ProxyCtx) (*tls.Config, error) {
		return tlsConfig(host, ctx)
	}
}

func (s *Server) setupLogging() {
	// Always set up request logging for persistence
	s.proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		if s.logger != nil {
			s.logger.Printf(">> %s %s", req.Method, req.URL)
		}

		// Capture request for logging
		entry, _ := s.reqLogger.LogRequest(req)
		ctx.UserData = entry

		return req, nil
	})

	s.proxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		if s.logger != nil && resp != nil {
			s.logger.Printf("<< %d %s", resp.StatusCode, ctx.Req.URL)
		}

		// Complete and persist log entry
		if entry, ok := ctx.UserData.(*RequestLog); ok {
			s.reqLogger.LogResponse(entry, resp, entry.Timestamp)
			if err := s.reqLogger.Log(entry); err != nil && s.logger != nil {
				s.logger.Printf("failed to write request log: %v", err)
			}
		}

		return resp
	})
}

func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("server already running")
	}

	// Try to listen on the configured port, fall back to next ports if busy
	var listener net.Listener
	var err error
	port := s.config.Port

	for i := 0; i < MaxPortRetries; i++ {
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		listener, err = net.Listen("tcp", addr)
		if err == nil {
			break
		}

		// Check if error is "address already in use"
		if !isAddrInUse(err) {
			return fmt.Errorf("failed to listen on %s: %w", addr, err)
		}

		// Try next port
		port++
	}

	if listener == nil {
		return fmt.Errorf("failed to find available port after %d attempts (tried %d-%d)",
			MaxPortRetries, s.config.Port, port-1)
	}

	// Update config with actual port used
	s.config.Port = port

	s.listener = listener
	s.server = &http.Server{
		Handler: s.proxy,
	}

	s.running = true

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			if s.logger != nil {
				s.logger.Printf("server error: %v", err)
			}
		}
	}()

	if s.logger != nil {
		s.logger.Printf("Proxy server started on %s", s.listener.Addr().String())
	}

	return nil
}

// isAddrInUse checks if the error is "address already in use"
func isAddrInUse(err error) bool {
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		var sysErr *os.SyscallError
		if errors.As(opErr.Err, &sysErr) {
			return errors.Is(sysErr.Err, syscall.EADDRINUSE)
		}
	}
	return false
}

func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false

	if s.listener != nil {
		_ = s.listener.Close()
	}

	s.wg.Wait()

	// Close request logger to flush remaining data
	if s.reqLogger != nil {
		_ = s.reqLogger.Close()
	}

	if s.logger != nil {
		s.logger.Printf("Proxy server stopped")
	}

	return nil
}

func (s *Server) Addr() string {
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

func (s *Server) Port() int {
	return s.config.Port
}

func (s *Server) CA() *CA {
	return s.ca
}

func (s *Server) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}
