package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/standardbeagle/claude-go"
)

func main() {
	ctx := context.Background()

	client := claude.New(&claude.Options{
		Debug:          false,
		Verbose:        true,
		PermissionMode: "bypassPermissions",
		Model:          "sonnet",
	})
	defer client.Close()

	fmt.Println("Claude Go Interactive Example")
	fmt.Println("Type 'quit' to exit, 'new' for new session")
	fmt.Println("===============================")

	var currentSession *claude.Session
	var sessionID string
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Error reading input: %v", err)
			continue
		}

		input = strings.TrimSpace(input)

		if input == "quit" {
			break
		}

		if input == "new" {
			if currentSession != nil {
				currentSession.Close()
			}
			currentSession = nil
			sessionID = ""
			fmt.Println("Ready for new session")
			continue
		}

		if input == "" {
			continue
		}

		if currentSession == nil {
			resp, err := client.Query(ctx, &claude.QueryRequest{
				Prompt: input,
			})
			if err != nil {
				log.Printf("Failed to create session: %v", err)
				continue
			}

			sessionID = resp.SessionID
			fmt.Printf("Created session: %s\n", sessionID)

			session, exists := client.GetSession(sessionID)
			if !exists {
				log.Printf("Session not found: %s", sessionID)
				continue
			}
			currentSession = session

			go func() {
				for err := range resp.Errors {
					log.Printf("Error: %v", err)
				}
			}()

			go func() {
				for msg := range resp.Messages {
					if msg.Type == "content" || msg.Type == "text" {
						fmt.Printf("\nClaude: %s\n> ", msg.Content)
					}
				}
			}()

		} else {
			if err := currentSession.SendMessage(input); err != nil {
				log.Printf("Failed to send message: %v", err)
				currentSession.Close()
				currentSession = nil
				sessionID = ""
				continue
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	if currentSession != nil {
		currentSession.Close()
	}

	fmt.Println("Goodbye!")
}
