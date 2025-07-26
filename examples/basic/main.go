package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/standardbeagle/claude-go"
)

func main() {
	ctx := context.Background()

	client := claude.New(&claude.Options{
		Debug:          true,
		Verbose:        true,
		PermissionMode: "bypassPermissions",
	})
	defer client.Close()

	resp, err := client.Query(ctx, &claude.QueryRequest{
		Prompt: "Write a simple hello world function in Go",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Session ID: %s\n", resp.SessionID)

	go func() {
		for err := range resp.Errors {
			log.Printf("Error: %v", err)
		}
	}()

	timeout := time.After(30 * time.Second)

	for {
		select {
		case msg, ok := <-resp.Messages:
			if !ok {
				fmt.Println("Message channel closed")
				return
			}

			fmt.Printf("[%s] %s: %s\n",
				msg.Timestamp.Format("15:04:05"),
				msg.Type,
				msg.Content)

		case <-timeout:
			fmt.Println("Timeout reached")
			return
		}
	}
}
