package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// TestHTTPSenderBatching проверяет корректность батчинга при разных сценариях
func TestHTTPSenderBatching(t *testing.T) {
	// Общая настройка для всех подтестов
	setup := func(batchSize int, batchTimeout time.Duration) (*httptest.Server, *httpSender, *batchRecorder) {
		recorder := &batchRecorder{}
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				PingResults []PingResult `json:"ping_results"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatal("decode error:", err)
			}
			recorder.add(req.PingResults)
			w.WriteHeader(http.StatusOK)
		}))

		sender := newHTTPSender(ts.URL, batchSize, batchTimeout)
		return ts, sender, recorder
	}

	t.Run("batch is sent when full", func(t *testing.T) {
		batchSize := 3
		batchTimeout := 10 * time.Millisecond
		ts, sender, recorder := setup(batchSize, batchTimeout)
		defer ts.Close()

		// Отправляем ровно batchSize результатов
		for i := 0; i < batchSize; i++ {
			sender.Send(PingResult{HostID: i})
		}

		// Ждем батч с таймаутом
		recorder.waitBatches(t, 1, batchTimeout/2)
		if len(recorder.batches[0]) != batchSize {
			t.Errorf("expected batch of %d elements, received %d", batchSize, len(recorder.batches[0]))
		}
	})

	t.Run("batch is sent on timeout", func(t *testing.T) {
		batchSize := 3
		batchTimeout := 10 * time.Millisecond
		ts, sender, recorder := setup(batchSize, batchTimeout)
		defer ts.Close()

		// Отправляем 2 элемента и ждем таймаута
		sender.Send(PingResult{HostID: 1})
		sender.Send(PingResult{HostID: 2})

		recorder.waitBatches(t, 1, batchTimeout*3/2)
		if len(recorder.batches[0]) != 2 {
			t.Errorf("expected batch of %d elements, received %d", batchSize, len(recorder.batches[0]))
		}
	})

	t.Run("multiple data packages", func(t *testing.T) {
		batchSize := 3
		batchTimeout := 10 * time.Millisecond
		ts, sender, recorder := setup(batchSize, batchTimeout)
		defer ts.Close()

		// Первый пакет: 2 элемента + таймаут
		sender.Send(PingResult{HostID: 1})
		sender.Send(PingResult{HostID: 2})
		recorder.waitBatches(t, 1, batchTimeout*3/2)

		// Второй пакет: полный батч
		for i := 0; i < batchSize; i++ {
			sender.Send(PingResult{HostID: i + 2})
		}
		recorder.waitBatches(t, 2, batchTimeout)

		// Проверяем размеры батчей
		if len(recorder.batches) != 2 {
			t.Fatalf("expected 2 batches, received %d", len(recorder.batches))
		}
		if len(recorder.batches[0]) != 2 {
			t.Errorf("first batch: expected 2 elements, received %d", len(recorder.batches[0]))
		}
		if len(recorder.batches[1]) != batchSize {
			t.Errorf("second batch: expected %d elements, received %d", batchSize, len(recorder.batches[1]))
		}
	})

	t.Run("large data package", func(t *testing.T) {
		batchSize := 3
		batchTimeout := 10 * time.Millisecond
		ts, sender, recorder := setup(batchSize, batchTimeout)
		defer ts.Close()

		// Отправляем batchSize+2 элементов подряд
		for i := 0; i < batchSize+2; i++ {
			sender.Send(PingResult{HostID: i})
		}

		// Ожидаем 2 батча: batchSize + 2
		recorder.waitBatches(t, 2, batchTimeout*3/2)
		if len(recorder.batches) != 2 {
			t.Fatalf("expected 2 batches, received %d", len(recorder.batches))
		}
		if len(recorder.batches[0]) != batchSize {
			t.Errorf("first batch: expected %d elements, received %d", batchSize, len(recorder.batches[0]))
		}
		if len(recorder.batches[1]) != 2 {
			t.Errorf("second batch: expected 2 elements, received %d", len(recorder.batches[1]))
		}
	})
}

// batchRecorder записывает полученные батчи
type batchRecorder struct {
	mu      sync.Mutex
	batches [][]PingResult
}

func (b *batchRecorder) add(batch []PingResult) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.batches = append(b.batches, batch)
}

// waitBatches ожидает указанное количество батчей с таймаутом
func (b *batchRecorder) waitBatches(t *testing.T, want int, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for {
		b.mu.Lock()
		if len(b.batches) >= want {
			b.mu.Unlock()
			return
		}
		b.mu.Unlock()

		if time.Now().After(deadline) {
			t.Fatalf("waiting time for batches has expired. Expected %b, received %d", want, len(b.batches))
		}
		time.Sleep(2 * time.Millisecond)
	}
}
