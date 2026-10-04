package subprocess

import (
	"os"
	"strings"
	"syscall"
	"time"
)

// isProcessAlreadyFinishedError checks if an error indicates the process has already terminated.
// This follows the Python SDK pattern of suppressing "process not found" type errors.
func isProcessAlreadyFinishedError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "process already finished") ||
		strings.Contains(errStr, "process already released") ||
		strings.Contains(errStr, "no child processes") ||
		strings.Contains(errStr, "signal: killed")
}

// terminateProcess implements the 5-second SIGTERM -> SIGKILL sequence
func (t *Transport) terminateProcess() error {
	if t.cmd == nil || t.cmd.Process == nil {
		return nil
	}

	// Send SIGTERM
	if err := t.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		// If process is already finished, that's success
		if isProcessAlreadyFinishedError(err) {
			return nil
		}
		// If SIGTERM fails for other reasons, try SIGKILL immediately
		killErr := t.cmd.Process.Kill()
		if killErr != nil && !isProcessAlreadyFinishedError(killErr) {
			return killErr
		}
		return nil // Don't return error for expected termination
	}

	// Wait exactly 5 seconds
	done := make(chan error, 1)
	// Capture cmd while we know it's valid to avoid data race
	cmd := t.cmd
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		// Normal termination or expected signals are not errors
		if err != nil {
			// Check if it's an expected exit signal
			if strings.Contains(err.Error(), "signal:") {
				return nil // Expected signal termination
			}
		}
		return err
	case <-time.After(terminationTimeoutSeconds * time.Second):
		// Force kill after 5 seconds
		if killErr := t.cmd.Process.Kill(); killErr != nil && !isProcessAlreadyFinishedError(killErr) {
			return killErr
		}
		// Wait for process to exit after kill
		<-done
		return nil
	case <-t.ctx.Done():
		// Context canceled - force kill immediately
		if killErr := t.cmd.Process.Kill(); killErr != nil && !isProcessAlreadyFinishedError(killErr) {
			return killErr
		}
		// Wait for process to exit after kill, but don't return context error
		// since this is normal cleanup behavior
		<-done
		return nil
	}
}

// abortConnect tears down a Connect that failed after the CLI started (the
// control protocol would not start, or the CLI exited or went silent before
// answering the handshake). Unlike Close it does not wait out a graceful
// SIGTERM: the process is killed at once. The order matters: the I/O
// goroutines must have returned before cleanup() releases the pipes they
// read, and the child must be reaped (Wait) so it does not linger as a
// zombie. It returns the child's Wait error (its exit status), which Connect
// adds to the error it reports when the CLI died before the handshake.
func (t *Transport) abortConnect() error {
	if t.protocol != nil {
		_ = t.protocol.Close()
		t.protocol = nil
	}
	if t.protocolAdapter != nil {
		_ = t.protocolAdapter.Close()
		t.protocolAdapter = nil
	}
	if t.cancel != nil {
		t.cancel()
	}
	if t.stdin != nil {
		_ = t.stdin.Close()
		t.stdin = nil
	}
	if t.cmd != nil && t.cmd.Process != nil {
		_ = t.cmd.Process.Kill() // best effort: it may already have exited
	}

	// The child is gone, so its stdout/stderr hit EOF and the goroutines
	// return. Bounded anyway: a grandchild can inherit and hold the pipes.
	done := make(chan struct{})
	go func() {
		t.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(terminationTimeoutSeconds * time.Second):
	}

	var waitErr error
	if t.cmd != nil && t.cmd.Process != nil {
		waitErr = t.cmd.Wait() // reap
	}
	t.cleanup()
	return waitErr
}

// cleanup cleans up all resources
func (t *Transport) cleanup() {
	if t.stdout != nil {
		_ = t.stdout.Close()
		t.stdout = nil
	}

	if t.stderrPipe != nil {
		_ = t.stderrPipe.Close()
		t.stderrPipe = nil
	}

	if t.stderr != nil {
		// Graceful cleanup matching Python SDK pattern
		// Python: except Exception: pass
		_ = t.stderr.Close()
		_ = os.Remove(t.stderr.Name()) // Ignore cleanup errors
		t.stderr = nil
	}

	if t.mcpConfigFile != nil {
		// Clean up temporary MCP config file
		_ = t.mcpConfigFile.Close()
		_ = os.Remove(t.mcpConfigFile.Name()) // Ignore cleanup errors
		t.mcpConfigFile = nil
	}

	// Reset state
	t.cmd = nil
}
