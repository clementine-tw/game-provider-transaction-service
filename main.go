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

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {

	// load enviroment vars

	godotenv.Load()

	// logger

	stdoutHandler := slog.NewJSONHandler(os.Stdout, nil)
	multiHandler := slog.NewMultiHandler(stdoutHandler)
	logger := slog.New(multiHandler)

	// database
	config, err := pgxpool.ParseConfig(os.Getenv("POSTGRES_URL"))
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		logger.Error("connect to db failed", "error", err)
		return
	}
	defer pool.Close()
	if err := pool.Ping(context.Background()); err != nil {
		logger.Error("ping db failed", "error", err)
		return
	}

	// routes

	mux := http.NewServeMux()

	// healthy
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("healthy"))
	})

	// debug long work
	mux.HandleFunc("GET /long", func(w http.ResponseWriter, r *http.Request) {
		logger.Debug("long work")
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
		logger.Error("shutdown server error", "error", err)
	}
	logger.Info("Server closed")
}
