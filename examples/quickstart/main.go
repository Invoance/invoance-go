// Command quickstart shows the core Invoance SDK flow: validate the key,
// ingest an event, and read it back.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Invoance/invoance-go"
)

func main() {
	ctx := context.Background()

	// Reads INVOANCE_API_KEY (and optional INVOANCE_BASE_URL) from the env.
	client, err := invoance.New()
	if err != nil {
		log.Fatal(err)
	}

	if res := client.Validate(ctx); !res.Valid {
		log.Fatalf("Invoance key invalid: %s (base %s)", res.Reason, res.BaseURL)
	}

	event, err := client.Events.Ingest(ctx, invoance.IngestEventParams{
		EventType: "user.login",
		Payload:   map[string]any{"user_id": "u_42"},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("ingested:", event.EventID)

	full, err := client.Events.Get(ctx, event.EventID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("payload hash:", full.PayloadHash)
}
