package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type httpSender struct {
	done  chan struct{}
	url   string
	c     chan PingResult
	batch []PingResult
}

func newHTTPSender(url string, batchSize int, batchTimeout time.Duration) *httpSender {
	snd := &httpSender{
		done:  make(chan struct{}),
		url:   url,
		c:     make(chan PingResult),
		batch: make([]PingResult, 0, batchSize),
	}
	go snd.serve(batchTimeout)
	return snd
}

func (s *httpSender) Send(result PingResult) {
	s.c <- result
}

func (s *httpSender) Close() {
	close(s.c)
	<-s.done
}

func (s *httpSender) serve(batchTimeout time.Duration) {
	defer close(s.done)

	tm := time.NewTimer(0)

	for {
		result, ok := <-s.c
		if !ok {
			return
		}
		s.batch = append(s.batch[:0], result)

		tm.Reset(batchTimeout)
	waitLoop:
		for len(s.batch) < cap(s.batch) {
			select {
			case <-tm.C:
				break waitLoop
			case result, ok := <-s.c:
				if !ok {
					break waitLoop
				}
				s.batch = append(s.batch, result)
			}
		}

		s.sendBatch()
	}
}

func (s *httpSender) sendBatch() {
	req := struct {
		PingResults []PingResult `json:"ping_results"`
	}{
		s.batch,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		slog.Error("can't marshal request", "error", err, "req", req)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.url, bytes.NewBuffer(jsonData))
	if err != nil {
		slog.Error("can't create http request", "error", err)
		return
	}

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		slog.Error("can't do http request", "error", err)
		return
	}
	defer func() {
		io.Copy(io.Discard, httpResp.Body)
		httpResp.Body.Close()
	}()

	if httpResp.StatusCode >= 400 {
		body, _ := io.ReadAll(httpResp.Body)
		slog.Error("remote return error", "statusCode", httpResp.StatusCode, "body", unsafeString(body))
		return
	}
}
