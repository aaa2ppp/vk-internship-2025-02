package main

import (
	"context"
	"log/slog"
	"slices"
	"sync"
)

type cacheRepo interface {
	GetHosts(ctx context.Context) ([]Host, error)
	GetLastSuccessPingResults(ctx context.Context) ([]PingResult, error)
	AddPingResults(ctx context.Context, results []PingResult) error
}

type cache struct {
	repo  cacheRepo
	mu    sync.Mutex
	data  []PingResult
	index map[int]int
}

func NewCache(repo cacheRepo) *cache {
	return &cache{repo: repo}
}

func (ca *cache) Init(ctx context.Context) error {
	hosts, err := ca.repo.GetHosts(ctx)
	if err != nil {
		return err
	}

	data := make([]PingResult, len(hosts))
	index := make(map[int]int, len(hosts))

	for i, host := range hosts {
		data[i] = PingResult{HostID: host.ID, HostName: host.Name}
		index[host.ID] = i
	}

	results, err := ca.repo.GetLastSuccessPingResults(ctx)
	if err != nil {
		return err
	}

	for i := range results {
		src := &results[i]
		dst := &data[index[src.HostID]]
		ca.copyPingResult(src, dst)
	}

	ca.data = data
	ca.index = index
	return nil
}

func (ca *cache) getLogger(ctx context.Context, op string) *slog.Logger {
	return GetLoggerFromContext(ctx).With("op", "cache."+op)
}

func (ca *cache) copyPingResult(dst, src *PingResult) {
	dst.IP = src.IP
	dst.Time = src.Time
	dst.Rtt = src.Rtt
	dst.Success = src.Success
}

func (ca *cache) GetLastSuccessPingResults(ctx context.Context) ([]PingResult, error) {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	if ca.data == nil {
		if err := ca.Init(ctx); err != nil {
			return nil, errInternalError
		}
	}

	log := ca.getLogger(ctx, "GetLastSuccessPingResults")
	log.Debug("", "ca.data", ca.data)

	return slices.Clone(ca.data), nil
}

func (ca *cache) AddPingResults(ctx context.Context, results []PingResult) error {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	if ca.data == nil {
		if err := ca.Init(ctx); err != nil {
			return errInternalError
		}
	}

	for i := range results {
		src := &results[i]
		j, ok := ca.index[src.HostID]
		if !ok {
			log := ca.getLogger(ctx, "AddPingResults")
			log.Error("host id not found in cache", "host", *src)
			return errBadRequest
		}
		dst := &ca.data[j]
		ca.copyPingResult(dst, src)
	}

	return ca.repo.AddPingResults(ctx, results)
}
