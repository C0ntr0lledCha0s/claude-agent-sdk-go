package subprocess

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/severity1/claude-agent-sdk-go/internal/shared"
)

// TestConnectCLIExitsBeforeHandshake covers a CLI that exits before answering
// the control-protocol handshake (a crash, a broken install, a missing
// binary behind a wrapper). Connect must fail cleanly: no panic on the stdout
// goroutine, no data race on the pipes, and nothing left running. Before the
// fix, the failure path closed and nilled t.stdout while handleStdout was
// still reading it, which raced (go test -race) and could panic in
// bufio.Scanner.Scan on a nil reader. It loops because the panic is a race.
func TestConnectCLIExitsBeforeHandshake(t *testing.T) {
	if runtime.GOOS == windowsOS {
		t.Skip("uses a POSIX shell script as the fake CLI")
	}
	fake := filepath.Join(t.TempDir(), "claude-exits")
	script := []byte("#!/bin/sh\necho 'boom' >&2\nexit 127\n")
	err := os.WriteFile(fake, script, 0o755) // #nosec G306 - Test script needs to be executable
	if err != nil {
		t.Fatal(err)
	}

	for _, withStderr := range []bool{false, true} {
		for i := 0; i < 60; i++ {
			opts := &shared.Options{EnableFileCheckpointing: true} // forces the handshake
			if withStderr {
				opts.StderrCallback = func(string) {}
			}
			tr := New(fake, opts, false, "sdk-go")
			// Short, varied deadlines make Connect fail while handleStdout is
			// still starting — the window the race needs.
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(1+i%20)*time.Millisecond)
			err := tr.Connect(ctx)
			cancel()
			if err == nil {
				t.Fatal("Connect succeeded against a CLI that exits at once")
			}
			if tr.IsConnected() {
				t.Fatal("transport reports connected after a failed Connect")
			}
			// A failed Connect leaves the transport closed; Close is a no-op.
			if cerr := tr.Close(); cerr != nil {
				t.Fatalf("Close after a failed Connect: %v", cerr)
			}
		}
	}
}

// TestConnectFailsFastWhenCLIExits checks the handshake notices the CLI's
// stdout closing: Connect must fail well inside a long caller deadline (it
// used to wait out the whole init timeout), say why, and carry the CLI's exit
// status.
func TestConnectFailsFastWhenCLIExits(t *testing.T) {
	if runtime.GOOS == windowsOS {
		t.Skip("uses a POSIX shell script as the fake CLI")
	}
	fake := filepath.Join(t.TempDir(), "claude-exits")
	err := os.WriteFile(fake, []byte("#!/bin/sh\nexit 127\n"), 0o755) // #nosec G306 - Test script needs to be executable
	if err != nil {
		t.Fatal(err)
	}

	tr := New(fake, &shared.Options{EnableFileCheckpointing: true}, false, "sdk-go")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	start := time.Now()
	err = tr.Connect(ctx)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Connect succeeded against a CLI that exits at once")
	}
	if elapsed > 5*time.Second {
		t.Fatalf("Connect took %v to notice the CLI had exited", elapsed)
	}
	if !errors.Is(err, errCLIExitedBeforeHandshake) {
		t.Fatalf("error does not say the CLI exited before the handshake: %v", err)
	}
	if !strings.Contains(err.Error(), "exit status 127") {
		t.Fatalf("error does not carry the CLI's exit status: %v", err)
	}
}
