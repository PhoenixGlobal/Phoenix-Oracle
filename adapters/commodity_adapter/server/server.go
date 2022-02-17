package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	config "github.com/chutommy/market-info/config"
	handlers "github.com/chutommy/market-info/handlers"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// Server controls the web server behaviour.
type Server struct {
	srv *http.Server
	h   *handlers.Handler
}

// New is a constructor for the Server.
func New() *Server {
	return &Server{}
}

// Set applies the configuration, set up the services and the server.
func (s *Server) Set(cfg *config.Config) error {

	// create a new router
	r := gin.New()

	// apply crach free middleware
	r.Use(gin.Recovery())

	// apply custom logging
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// your custom format
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s\" %s %s\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	}))
	// log to file and os.Stdout
	f, _ := os.Create("server.log")
	gin.DefaultWriter = io.MultiWriter(f, os.Stdout)

	// get Handler
	s.h = handlers.New()
	err := s.h.Init(cfg.CommodityServiceTarget, cfg.CurrencyServiceTarget, cfg.CryptoServiceTarget)
	if err != nil {
		return errors.Wrap(err, "setting up the handler")
	}

	// apply routing
	SetRoutes(r, s.h)

	// set the server properties
	s.srv = &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.APIPort),
		Handler:           r,
		ReadTimeout:       800 * time.Millisecond,
		ReadHeaderTimeout: 500 * time.Millisecond,
		WriteTimeout:      1000 * time.Millisecond,
		IdleTimeout:       10 * time.Second,
		MaxHeaderBytes:    http.DefaultMaxHeaderBytes,
	}

	return nil
}

// Start starts the server.
func (s *Server) Start() error {
	return s.srv.ListenAndServe()
}

// Stop closes all connections and dials.
func (s *Server) Stop() error {

	// stop the handler's services
	err := s.h.Stop()
	if err != nil {
		return errors.Wrap(err, "stopping handler's services")
	}

	// gracefully shutdown
	timeout, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	err = s.srv.Shutdown(timeout)
	if err != nil {
		return errors.Wrap(err, "failed to gracefully shutdown")
	}
	return nil
}
