package control

import (
	"errors"
	"testing"
	"time"
)

// TestFailPendingRequestsWakesWaiters verifies the stream's terminal error
// reaches every control request waiting for a response at once, instead of
// each running into its own timeout (Python: the reader's error is set on
// every pending request).
func TestFailPendingRequestsWakesWaiters(t *testing.T) {
	tests := []struct {
		name    string
		pending int
	}{
		{"one_pending_request", 1},
		{"several_pending_requests", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := setupControlTestContext(t, 30*time.Second)
			defer cancel()

			transport := newControlMockTransport()
			protocol := NewProtocol(transport)
			assertControlNoError(t, protocol.Start(ctx))
			defer func() { _ = protocol.Close() }()
			markInitialized(protocol)

			results := make(chan error, tt.pending)
			for i := 0; i < tt.pending; i++ {
				go func() { results <- protocol.Interrupt(ctx) }()
			}
			waitForWrittenRequests(t, transport, tt.pending)

			streamErr := errors.New("CLI exited")
			protocol.FailPendingRequests(streamErr)

			for i := 0; i < tt.pending; i++ {
				select {
				case err := <-results:
					if !errors.Is(err, streamErr) {
						t.Errorf("pending request error = %v, want the stream error", err)
					}
				case <-time.After(closeWakeBudget):
					t.Fatalf("request %d of %d still pending after FailPendingRequests", i+1, tt.pending)
				}
			}
		})
	}
}

// TestFailPendingRequestsLeavesInitializeToInitErr verifies a request still
// waiting for the initialize handshake is not failed here: the init watcher
// and the error-result routing already fail it through HandleControlInitErr.
func TestFailPendingRequestsLeavesInitializeToInitErr(t *testing.T) {
	ctx, cancel := setupControlTestContext(t, 30*time.Second)
	defer cancel()

	transport := newControlMockTransport()
	protocol := NewProtocol(transport)
	assertControlNoError(t, protocol.Start(ctx))
	defer func() { _ = protocol.Close() }()

	result := make(chan error, 1)
	go func() {
		_, err := protocol.Initialize(ctx)
		result <- err
	}()
	waitForWrittenRequests(t, transport, 1)

	protocol.FailPendingRequests(errors.New("CLI exited"))
	select {
	case err := <-result:
		t.Fatalf("Initialize returned %v, want it to keep waiting", err)
	case <-time.After(100 * time.Millisecond):
	}

	initErr := errors.New("init failed")
	protocol.HandleControlInitErr(initErr)
	select {
	case err := <-result:
		if !errors.Is(err, initErr) {
			t.Errorf("Initialize error = %v, want the init error", err)
		}
	case <-time.After(closeWakeBudget):
		t.Fatal("Initialize still pending after HandleControlInitErr")
	}
}

func markInitialized(p *Protocol) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.initialized = true
}
