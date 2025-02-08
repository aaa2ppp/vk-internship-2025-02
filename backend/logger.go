package main

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"runtime/debug"
)

type loggerContextID struct{}

func ContextWithLogger(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerContextID{}, log)
}

func GetLoggerFromContext(ctx context.Context) *slog.Logger {
	log := ctx.Value(loggerContextID{})
	if log == nil {
		return slog.Default()
	}
	return log.(*slog.Logger)
}

// Logging middleware
func Logging(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log := slog.Default().With("httpReqID", rand.Uint64())

		url := r.URL.String()
		log.Debug("http request begin", "remoteAddr", r.RemoteAddr, "method", r.Method, "url", url)

		w = newWriteHeaderHook(w, func(statusCode int) {
			log.Debug("http request end", "statusCode", statusCode)
		})

		ctx := ContextWithLogger(r.Context(), log)
		r = r.WithContext(ctx)

		defer func() {
			if p := recover(); p != nil {
				log.Error("*** panic recovered ***", "panic", p, "stack", debug.Stack())
			}
		}()

		h.ServeHTTP(w, r)
	}
}

type writeHeaderHook struct {
	http.ResponseWriter
	hook func(statusCode int)
	flag bool // need use atomic.Bool for thread safety
}

func newWriteHeaderHook(w http.ResponseWriter, hook func(statusCode int)) *writeHeaderHook {
	return &writeHeaderHook{
		ResponseWriter: w,
		hook:           hook,
	}
}

func (rw *writeHeaderHook) WriteHeader(statusCode int) {
	if !rw.flag {
		rw.flag = true
		rw.hook(statusCode)
		rw.ResponseWriter.WriteHeader(statusCode)
	}
}

func (rw *writeHeaderHook) Write(b []byte) (int, error) {
	rw.WriteHeader(http.StatusOK)
	return rw.ResponseWriter.Write(b)
}
