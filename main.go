package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	// logger

	stdoutHandler := slog.NewJSONHandler(os.Stdout, nil)
	multiHandler := slog.NewMultiHandler(stdoutHandler)
	logger := slog.New(multiHandler)

	// routes

	mux := http.NewServeMux()

	// healthy
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("healthy"))
	})

	// debug long work
	mux.HandleFunc("GET /long", func(w http.ResponseWriter, r *http.Request) {
		slog.Debug("long work")
		time.Sleep(10 * time.Second)
	})

	serv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	logger.Info("Server start listening at port 8080")
	go func() {
		if err := serv.ListenAndServe(); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				panic(err)
			}
		}
	}()
	shutdown := make(chan os.Signal)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	<-shutdown

	logger.Info("Stopping server")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	if err := serv.Shutdown(ctx); err != nil {
		slog.Error("shutdown server error", "error", err)
	}
	logger.Info("Server closed")
}
