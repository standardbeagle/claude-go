package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/standardbeagle/claude-go"
)

func main() {
	ctx := context.Background()

	client := claude.New(&claude.Options{
		Debug:          false,
		Verbose:        false,
		PermissionMode: "bypassPermissions",
	})
	defer client.Close()

	prompts := []string{
		"Write a function to calculate factorial",
		"Write a function to reverse a string",
		"Write a function to check if a number is prime",
	}

	var wg sync.WaitGroup

	for i, prompt := range prompts {
		wg.Add(1)

		go func(id int, p string) {
			defer wg.Done()

			resp, err := client.Query(ctx, &claude.QueryRequest{
				Prompt: p,
			})
			if err != nil {
				log.Printf("Query %d failed: %v", id, err)
				return
			}

			fmt.Printf("Started session %d: %s\n", id, resp.SessionID)

			go func() {
				for err := range resp.Errors {
					log.Printf("Session %d error: %v", id, err)
				}
			}()

			timeout := time.After(20 * time.Second)
			messageCount := 0

			for {
				select {
				case msg, ok := <-resp.Messages:
					if !ok {
						fmt.Printf("Session %d completed with %d messages\n", id, messageCount)
						return
					}

					messageCount++
					if msg.Type == "content" && len(msg.Content) > 100 {
						fmt.Printf("Session %d: [%s] %s: %.100s...\n",
							id, msg.Timestamp.Format("15:04:05"), msg.Type, msg.Content)
					} else {
						fmt.Printf("Session %d: [%s] %s: %s\n",
							id, msg.Timestamp.Format("15:04:05"), msg.Type, msg.Content)
					}

				case <-timeout:
					fmt.Printf("Session %d timed out\n", id)
					return
				}
			}
		}(i+1, prompt)
	}

	wg.Wait()
	fmt.Println("All sessions completed")
}
