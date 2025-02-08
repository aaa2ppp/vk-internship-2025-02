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
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
	url    string
	c      chan PingResult
	batch  []PingResult
}

func newHTTPSender(url string, n int) *httpSender {
	ctx, cancel := context.WithCancel(context.Background())
	snd := &httpSender{
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
		url:    url,
		c:      make(chan PingResult),
		batch:  make([]PingResult, n),
	}
	go snd.serve()
	return snd
}

func (s *httpSender) Send(result PingResult) {
	s.c <- result
}

func (s *httpSender) Close() {
	close(s.c)
	s.cancel()
	<-s.done
}

func (s *httpSender) serve() {
	defer close(s.done)

	tm := time.NewTimer(100500 * time.Second)
	tm.Stop()

	for {
		result, ok := <-s.c
		if !ok {
			return
		}
		s.batch = append(s.batch[:0], result)

		tm.Reset(sendBatchTimeout)

	waitLoop:
		for {
			select {
			case <-tm.C:
				break waitLoop
			case result, ok := <-s.c:
				if !ok {
					break waitLoop
				}
				s.batch = append(s.batch, result)
				if len(s.batch) == cap(s.batch) {
					break waitLoop
				}
			}
		}

		// if !tm.Stop() { // начиная с 1.23 в этом нет необходимости
		// 	<-tm.C
		// }

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

	httpReq, err := http.NewRequestWithContext(s.ctx, "POST", pingResultsURL, bytes.NewBuffer(jsonData))
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
		body, _ := io.ReadAll(httpReq.Body)
		slog.Error("remote return error", "statusCode", httpResp.StatusCode, "body", unsafeString(body))
		return
	}
}
