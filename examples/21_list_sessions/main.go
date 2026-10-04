// Package main demonstrates listing Claude Code sessions from disk,
// including git worktree support.
//
// Sessions are persisted as JSONL files under ~/.claude/projects/.
// ListSessions reads their metadata without needing a CLI connection.
//
// Usage:
//
//	go run main.go                     # List 10 most recent sessions across all projects
//	go run main.go /path/to/repo       # List sessions for a specific project (with worktrees)
//	go run main.go /path/to/repo false # Same, but without worktree expansion
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	claudecode "github.com/severity1/claude-agent-sdk-go"
)

func main() {
	fmt.Println("Claude Agent SDK - List Sessions Example")
	fmt.Println("=========================================")

	var opts []claudecode.SessionOption
	opts = append(opts, claudecode.WithSessionLimit(10))

	// If a directory argument is provided, scope to that project.
	if len(os.Args) > 1 {
		dir := os.Args[1]
		opts = append(opts, claudecode.WithSessionDirectory(dir))
		fmt.Printf("Directory: %s\n", dir)

		// Optional second arg: "false" to disable worktree expansion.
		if len(os.Args) > 2 && os.Args[2] == "false" {
			opts = append(opts, claudecode.WithIncludeWorktrees(false))
			fmt.Println("Worktrees: disabled")
		} else {
			fmt.Println("Worktrees: enabled (default)")
		}
	} else {
		fmt.Println("Directory: all projects")
	}

	sessions, err := claudecode.ListSessions(opts...)
	if err != nil {
		log.Fatalf("ListSessions failed: %v", err)
	}

	if len(sessions) == 0 {
		fmt.Println("\nNo sessions found. Run a Claude query first to create one.")
		return
	}

	fmt.Printf("\nFound %d session(s):\n\n", len(sessions))
	for i, s := range sessions {
		modified := time.UnixMilli(s.LastModified).Format("2006-01-02 15:04")

		// Summary is: custom title > AI title > first prompt > timestamp > session ID.
		summary := s.Summary
		if len(summary) > 80 {
			summary = summary[:77] + "..."
		}

		fmt.Printf("  %d. %s\n", i+1, summary)
		fmt.Printf("     ID: %s\n", s.SessionID)
		fmt.Printf("     Modified: %s\n", modified)
		if s.GitBranch != nil {
			fmt.Printf("     Branch: %s\n", *s.GitBranch)
		}
		if s.Cwd != nil {
			fmt.Printf("     Cwd: %s\n", *s.Cwd)
		}
		fmt.Println()
	}
}
