// Package main demonstrates reading session messages from disk.
//
// This example creates a real session via Query, then reads it back
// using GetSessionInfo and GetSessionMessages to show the typed
// content blocks (text, tool_use, tool_result, etc.).
//
// Run: go run main.go
// Requires: Claude CLI installed and authenticated.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	claudecode "github.com/severity1/claude-agent-sdk-go"
)

func main() {
	fmt.Println("Claude Agent SDK - Session Messages Example")
	fmt.Println("============================================")

	ctx := context.Background()

	// Step 1: Create a session by running a query.
	fmt.Println("\n1. Running a query to create a session...")
	sessionID, err := runQuery(ctx)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	fmt.Printf("   Session ID: %s\n", sessionID)

	// Step 2: Read session metadata.
	fmt.Println("\n2. Reading session info from disk...")
	info, err := claudecode.GetSessionInfo(sessionID)
	if err != nil {
		log.Fatalf("GetSessionInfo failed: %v", err)
	}
	if info == nil {
		log.Fatal("Session not found on disk")
	}

	fmt.Printf("   Summary: %s\n", info.Summary)
	if info.Cwd != nil {
		fmt.Printf("   Cwd: %s\n", *info.Cwd)
	}
	if info.CreatedAt != nil {
		fmt.Printf("   Created: %s\n", time.UnixMilli(*info.CreatedAt).Format(time.RFC3339))
	}

	// Step 3: Read messages.
	fmt.Println("\n3. Reading session messages...")
	msgs, err := claudecode.GetSessionMessages(sessionID)
	if err != nil {
		log.Fatalf("GetSessionMessages failed: %v", err)
	}

	fmt.Printf("   Found %d message(s):\n\n", len(msgs))
	for i, msg := range msgs {
		fmt.Printf("   [%d] type=%s uuid=%s\n", i, msg.Type, msg.UUID)

		mc := msg.Content
		if mc == nil {
			continue
		}

		switch mc.Kind {
		case claudecode.SessionContentTypeString:
			preview := mc.String
			if len(preview) > 100 {
				preview = preview[:97] + "..."
			}
			fmt.Printf("       content (string): %q\n", preview)

		case claudecode.SessionContentTypeBlocks:
			for j, block := range mc.Blocks {
				fmt.Printf("       block[%d] type=%s", j, block.Type)
				switch block.Type {
				case "text":
					text := block.Text
					if len(text) > 80 {
						text = text[:77] + "..."
					}
					fmt.Printf(" text=%q", text)
				case "tool_use":
					fmt.Printf(" name=%s id=%s", block.Name, block.ID)
				case "tool_result":
					fmt.Printf(" tool_use_id=%s", block.ToolUseID)
				case "thinking":
					thinking := block.Thinking
					if len(thinking) > 60 {
						thinking = thinking[:57] + "..."
					}
					fmt.Printf(" thinking=%q", thinking)
				case "image":
					if block.Source != nil {
						fmt.Printf(" media_type=%v", block.Source["media_type"])
					}
				default:
					fmt.Printf(" (unknown type, raw keys: %v)", mapKeys(block.Raw))
				}
				fmt.Println()
			}
		}
	}

	fmt.Println("\nDone!")
}

func runQuery(ctx context.Context) (string, error) {
	iter, err := claudecode.Query(ctx, "Say hello briefly.",
		claudecode.WithMaxTurns(1),
		claudecode.WithPermissionMode(claudecode.PermissionModePlan),
	)
	if err != nil {
		return "", err
	}
	defer iter.Close()

	var sessionID string
	for {
		msg, err := iter.Next(ctx)
		if err != nil {
			if errors.Is(err, claudecode.ErrNoMoreMessages) {
				break
			}
			return "", err
		}
		if result, ok := msg.(*claudecode.ResultMessage); ok {
			sessionID = result.SessionID
		}
	}
	if sessionID == "" {
		return "", fmt.Errorf("no session ID in result")
	}
	return sessionID, nil
}

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
