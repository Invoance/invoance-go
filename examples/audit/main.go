// Command audit registers an org, appends a signed audit event, and verifies
// the returned event offline.
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

	orgID := "org_acme"
	if _, err := client.Audit.Orgs.Create(ctx, invoance.CreateAuditOrgParams{
		OrganizationID: orgID,
		Name:           "Acme Inc.",
	}); err != nil {
		log.Fatal(err)
	}

	if _, err := client.Audit.Events.Ingest(ctx, invoance.IngestAuditEventParams{
		OrganizationID: orgID,
		Action:         "invoice.approved",
		Actor:          map[string]any{"type": "user", "id": "u_1"},
		Targets:        []map[string]any{{"type": "invoice", "id": "inv_42"}},
		Metadata:       map[string]any{"amount": 1200},
	}); err != nil {
		log.Fatal(err)
	}

	list, err := client.Audit.Events.List(ctx, invoance.ListAuditEventsParams{
		OrganizationID: orgID,
	})
	if err != nil {
		log.Fatal(err)
	}

	for _, ev := range list.Events {
		// Offline verify against the event-embedded key. Pin the tenant's
		// registered key for a real tamper guarantee.
		res := invoance.VerifyAuditEventStruct(ev, nil)
		fmt.Printf("%s valid=%v reason=%q\n", ev.ID, res.Valid, res.Reason)
	}
}
