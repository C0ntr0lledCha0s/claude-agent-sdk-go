package control

import (
	"context"
	"errors"
	"testing"
	"time"
)

// closeWakeBudget is how long a request may take to fail after Close. The
// requests under test wait for a 5 second timeout otherwise.
const closeWakeBudget = time.Second

// waitForWrittenRequests blocks until the mock transport has been handed n
// requests, so every request is known to be waiting for its response.
func waitForWrittenRequests(t *testing.T, transport *controlMockTransport, n int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		transport.mu.Lock()
		written := len(transport.writtenData)
		transport.mu.Unlock()
		if written >= n {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("only %d of %d requests were written", written, n)
		}
		time.Sleep(time.Millisecond)
	}
}

// TestCloseFailsPendingControlRequests verifies Close wakes every control
// request still waiting for its response instead of leaving it to run into its
// own timeout (a request in flight when the client disconnects).
func TestCloseFailsPendingControlRequests(t *testing.T) {
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

			results := make(chan error, tt.pending)
			for i := 0; i < tt.pending; i++ {
				go func() { results <- protocol.Interrupt(ctx) }()
			}
			waitForWrittenRequests(t, transport, tt.pending)

			start := time.Now()
			assertControlNoError(t, protocol.Close())

			for i := 0; i < tt.pending; i++ {
				select {
				case err := <-results:
					if !errors.Is(err, ErrProtocolClosed) {
						t.Errorf("pending request error = %v, want ErrProtocolClosed", err)
					}
				case <-time.After(closeWakeBudget):
					t.Fatalf("request %d of %d still pending %v after Close", i+1, tt.pending, time.Since(start))
				}
			}
		})
	}
}

// TestControlRequestAfterCloseFails verifies a request sent after Close fails
// at once and is never written to the transport.
func TestControlRequestAfterCloseFails(t *testing.T) {
	model := testModelSonnet
	tests := []struct {
		name string
		send func(ctx context.Context, p *Protocol) error
	}{
		{"interrupt", func(ctx context.Context, p *Protocol) error { return p.Interrupt(ctx) }},
		{"set_model", func(ctx context.Context, p *Protocol) error { return p.SetModel(ctx, &model) }},
		{"get_mcp_status", func(ctx context.Context, p *Protocol) error { _, err := p.GetMcpStatus(ctx); return err }},
		{"initialize", func(ctx context.Context, p *Protocol) error { _, err := p.Initialize(ctx); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := setupControlTestContext(t, 30*time.Second)
			defer cancel()

			transport := newControlMockTransport()
			protocol := NewProtocol(transport)
			assertControlNoError(t, protocol.Start(ctx))
			assertControlNoError(t, protocol.Close())

			start := time.Now()
			err := tt.send(ctx, protocol)
			if !errors.Is(err, ErrProtocolClosed) {
				t.Errorf("request after Close error = %v, want ErrProtocolClosed", err)
			}
			if elapsed := time.Since(start); elapsed > closeWakeBudget {
				t.Errorf("request after Close took %v, want it to fail at once", elapsed)
			}

			transport.mu.Lock()
			defer transport.mu.Unlock()
			if len(transport.writtenData) != 0 {
				t.Errorf("wrote %d request(s) after Close: %q", len(transport.writtenData), transport.writtenData)
			}
		})
	}
}
