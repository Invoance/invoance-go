// Command traces creates a trace, attaches an event, seals it, and exports
// the proof bundle.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Invoance/invoance-go"
)

func main() {
	ctx := context.Background()
	client, err := invoance.New()
	if err != nil {
		log.Fatal(err)
	}

	trace, err := client.Traces.Create(ctx, invoance.CreateTraceParams{
		Label:    "onboarding-run-42",
		Metadata: map[string]any{"customer": "acme"},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("trace:", trace.TraceID)

	if _, err := client.Events.Ingest(ctx, invoance.IngestEventParams{
		EventType: "step.completed",
		Payload:   map[string]any{"step": "kyc"},
		TraceID:   trace.TraceID,
	}); err != nil {
		log.Fatal(err)
	}

	sealed, err := client.Traces.Seal(ctx, trace.TraceID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("seal status:", sealed.Status)

	// Proof export is available once the trace is fully sealed.
	proof, err := client.Traces.Proof(ctx, trace.TraceID)
	if err != nil {
		fmt.Println("proof not ready yet:", err)
		return
	}
	fmt.Println("composite hash:", proof.CompositeHash)
}
