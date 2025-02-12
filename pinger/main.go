package main

import (
	"cmp"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
	"unsafe"

	probing "github.com/prometheus-community/pro-bing"
)

const (
	// TODO: to config
	shutdownTimeout  = 30 * time.Second
	backendUpTimeout = 30 * time.Second

	batchTimeout   = 10 * time.Millisecond
	baseURL        = "http://backend:8080"
	pingResultsURL = baseURL + "/ping-results"
	hostsURL       = baseURL + "/hosts"
	pingURL        = baseURL + "/ping"
)

var (
	logLevel     = slog.LevelInfo
	pingInterval = 10 * time.Second
)

func main() {
	if _, ok := os.LookupEnv("DEBUG"); ok {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))

	// TODO: to config
	if s, ok := os.LookupEnv("PING_INTERVAL"); ok {
		if v, err := time.ParseDuration(s); err != nil {
			slog.Warn("can't parse PING_INTERVAL", "PING_INTERVAL", s)
		} else {
			pingInterval = v
		}
	}

	slog.Info("wait backend up...", "timeout", backendUpTimeout)
	if err := waitBackend(backendUpTimeout); err != nil {
		slog.Error("backend up timeout expired", "lastError", err)
		os.Exit(1)
	}

	hosts, err := getHosts()
	if err != nil {
		slog.Error("can't get hosts", "error", err)
		os.Exit(1)
	}
	if len(hosts) == 0 {
		slog.Error("nothing to ping")
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sender := newHTTPSender(pingResultsURL, len(hosts), batchTimeout)

	var wg sync.WaitGroup
	wg.Add(len(hosts))

	for i := range hosts {
		host := hosts[i]
		go func() {
			defer wg.Done()
			pingLoop(ctx, host, pingInterval, sender)
		}()
	}

	slog.Info("pinger is started, ping-pong begins...", "interval", pingInterval)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	signal := <-c

	slog.Info("shutdown by signal", "signal", signal, "timeout", shutdownTimeout)
	time.AfterFunc(shutdownTimeout, func() {
		slog.Error("shutdown timeout expired")
		os.Exit(1)
	})
	cancel()

	wg.Wait()
	sender.Close()
	slog.Info("pinger stopped")
}

func waitBackend(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	tm := time.NewTimer(0)

	var lastErr error
	for interval := 1 * time.Second; ; interval *= 2 {
		err := httpPing(ctx, pingURL)
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return cmp.Or(lastErr, err)
		}
		lastErr = err

		if interval > 30*time.Second {
			interval = 30
		}

		tm.Reset(interval)
		select {
		case <-tm.C:
		case <-ctx.Done():
			lastCtx, lastCancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer lastCancel()
			return httpPing(lastCtx, pingURL)
		}
	}
}

func httpPing(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return nil
}

func getHosts() ([]Host, error) {
	resp, err := http.Get(hostsURL)
	if err != nil {
		return nil, err
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var request = struct {
		Hosts []Host
	}{}
	if err := json.Unmarshal(buf, &request); err != nil {
		slog.Error("can't unmarshal body", "body", unsafeString(buf))
		return nil, err
	}

	return request.Hosts, nil
}

type sender interface {
	Send(result PingResult)
}

func pingLoop(ctx context.Context, host Host, interval time.Duration, snd sender) {
	pinger, err := probing.NewPinger(host.Name)
	if err != nil {
		slog.Error("can't create pinger", "error", err, "host", host.Name)
		return
	}

	pinger.Interval = interval
	pinger.RecordRtts = false
	pinger.RecordTTLs = false

	pinger.OnRecv = func(pkt *probing.Packet) {
		result := PingResult{
			HostID:  host.ID,
			IP:      pkt.Addr,
			Time:    time.Now(),
			Rtt:     pkt.Rtt,
			Success: true,
		}
		snd.Send(result)
	}

	pinger.RunWithContext(ctx)
}

func unsafeString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}
