package main

import (
	"cmp"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/lib/pq"
)

const (
	// TODO: to config
	readTimeout     = 10 * time.Second
	writeTimeout    = 10 * time.Second
	shutdownTimeout = 30 * time.Second
	dbHost          = "db"
	dbName          = "monitoring"
	dbUser          = "postgres"
	dbPassword      = "postgres"
	dbUpTimeout     = 30 * time.Second
)

var (
	logLevel = slog.LevelInfo
)

func main() {
	if _, ok := os.LookupEnv("DEBUG"); ok {
		logLevel = slog.LevelDebug
	}
	os.Exit(run())
}

func run() int {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))

	// TODO: load config

	db, err := openDB()
	if err != nil {
		slog.Error("can't open database", "error", err)
		return 1
	}
	defer db.Close()

	slog.Info("wait database up...", "timeout", dbUpTimeout)
	if err := waitDB(db, dbUpTimeout); err != nil {
		slog.Error("database up timeout expired", "lastErr", err)
		db.Close()
		return 1
	}

	// TODO: migrations up

	repo := NewRepo(db)

	if err := repo.AddHosts(context.Background(), getHostsFromEnv()); err != nil {
		return 1
	}
	cache := NewCache(repo)

	mux := http.NewServeMux()

	pong := func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("pong")) }

	mux.HandleFunc("GET  /ping", pong)
	mux.HandleFunc("GET  /hosts", getHostsHandler(repo))
	mux.HandleFunc("POST /ping-results", addPingResultHandler(cache))

	mux.HandleFunc("GET  /pub/ping", pong)
	mux.HandleFunc("GET  /pub/hosts", getHostsHandler(repo))
	mux.HandleFunc("GET  /pub/ping-results", getLastSuccessPingResultsHandler(cache))

	server := http.Server{
		Handler:      Logging(mux),
		Addr:         ":8080",
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	done := make(chan int)
	go func() {
		defer close(done)

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		signal := <-c

		slog.Info("shutdown by signal", "signal", signal, "timeout", shutdownTimeout)
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			slog.Error("can't shutdown http server", "error", err)
			done <- 1
		}
	}()

	slog.Info("http server startup", "addr", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("http server fail", "error", err)
		return 1
	}

	slog.Info("http server stopped")
	return <-done
}

func openDB() (*sql.DB, error) {
	connStr := fmt.Sprintf("host=%s dbname=%s user=%s password=%s sslmode=disable",
		dbHost, dbName, dbUser, dbPassword)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func waitDB(db *sql.DB, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	tm := time.NewTimer(100500 * time.Second)
	tm.Stop()

	var lastErr error
	for interval := 1 * time.Second; ; interval *= 2 {
		err := db.PingContext(ctx)
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return cmp.Or(lastErr, err)
		}
		lastErr = err

		tm.Reset(interval)
		select {
		case <-tm.C:
		case <-ctx.Done():
			return db.Ping()
		}
	}
}

func getHostsFromEnv() []string {
	hosts := strings.Split(os.Getenv("PING_HOSTS"), " ")

	// rermove empty items
	n := 0
	for i := range hosts {
		if hosts[i] == "" {
			continue
		}
		if n < i {
			hosts[n] = hosts[i]
		}
		n++
	}
	return hosts[:n]
}
