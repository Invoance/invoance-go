// Command attestations ingests an AI attestation and verifies its signature
// offline.
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

	att, err := client.Attestations.Ingest(ctx, invoance.IngestAttestationParams{
		Type:          "output",
		Input:         "Summarize this contract",
		Output:        "The contract states...",
		ModelProvider: "openai",
		ModelName:     "gpt-4o",
		ModelVersion:  "2025-01-01",
		Subject:       &invoance.AttestationSubject{UserID: "u_42"},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("attestation:", att.AttestationID)

	res, err := client.Attestations.VerifySignature(ctx, att.AttestationID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("signature valid:", res.Valid, res.Reason)
}
