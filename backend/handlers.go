package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
)

type handlerHelper struct {
	w   http.ResponseWriter
	r   *http.Request
	op  string
	log *slog.Logger
}

func newHandlerHelper(w http.ResponseWriter, r *http.Request, op string) *handlerHelper {
	return &handlerHelper{
		w:  w,
		r:  r,
		op: op,
	}
}

func (x *handlerHelper) Ctx() context.Context {
	return x.r.Context()
}

func (x *handlerHelper) Log() *slog.Logger {
	if x.log == nil {
		x.log = GetLoggerFromContext(x.Ctx()).With("op", "handler."+x.op)
	}
	return x.log
}

func (x handlerHelper) ReadBody(req any) error {
	body, err := io.ReadAll(x.r.Body)
	if err != nil {
		x.Log().Error("can't read body", "error", err)
		return errInternalError
	}
	if err := json.Unmarshal(body, req); err != nil {
		x.Log().Debug("can't unmarshal body", "error", err)
		return errBadRequest
	}
	return nil
}

func (x handlerHelper) WriteError(err error) {
	var httpError *httpError
	if errors.As(err, &httpError) {
		http.Error(x.w, httpError.Message, httpError.Status)
	} else {
		x.log.Warn("unhandled error expected", "error", err)
		http.Error(x.w, "internal error", 500)
	}
}

func (x handlerHelper) WriteResponse(resp any) {
	x.w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(x.w).Encode(resp); err != nil {
		x.log.Error("can't write response", "error", err)
	}
}

type getHostsResponse struct {
	Hosts []Host `json:"hosts"`
}

func getHostsHandler(s interface {
	GetHosts(ctx context.Context) ([]Host, error)
}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		x := newHandlerHelper(w, r, "GetHost")

		hosts, err := s.GetHosts(x.Ctx())
		if err != nil {
			x.WriteError(err)
			return
		}

		x.WriteResponse(getHostsResponse{
			Hosts: hosts,
		})
	}
}

type getPingResultsResponse struct {
	PingResults []PingResult `json:"ping_results"`
}

type lastSuccessPingResultGetter interface {
	GetLastSuccessPingResults(ctx context.Context) ([]PingResult, error)
}

func getLastSuccessPingResultsHandler(s lastSuccessPingResultGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		x := newHandlerHelper(w, r, "GetLastSuccessPingResults")

		results, err := s.GetLastSuccessPingResults(r.Context())
		if err != nil {
			x.WriteError(err)
			return
		}

		x.WriteResponse(getPingResultsResponse{
			PingResults: results,
		})
	}
}

type addPingResultRequest struct {
	PingResults []PingResult `json:"ping_results"`
}

type pingResultAdder interface {
	AddPingResults(ctx context.Context, results []PingResult) error
}

func addPingResultHandler(s pingResultAdder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		x := newHandlerHelper(w, r, "AddPingResults")

		var req addPingResultRequest
		if err := x.ReadBody(&req); err != nil {
			x.WriteError(err)
			return
		}

		if err := s.AddPingResults(r.Context(), req.PingResults); err != nil {
			x.WriteError(err)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}
