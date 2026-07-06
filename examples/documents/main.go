// Command documents anchors a document by its hash and verifies it.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

	docBytes := []byte("the quick brown fox")
	sum := sha256.Sum256(docBytes)
	hash := hex.EncodeToString(sum[:])

	anchored, err := client.Documents.Anchor(ctx, invoance.AnchorDocumentParams{
		DocumentHash: hash,
		DocumentRef:  "note.txt",
		Metadata:     map[string]any{"source": "example"},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("anchored:", anchored.EventID)

	v, err := client.Documents.Verify(ctx, anchored.EventID, invoance.VerifyDocumentParams{
		DocumentHash: hash,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("match:", v.MatchResult)
}
