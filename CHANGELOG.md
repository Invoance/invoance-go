# Changelog

All notable changes to this project are documented here. The format is based
on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `Audit.Orgs.Update` — rename an org (`PATCH /audit/orgs/{id}`). A nil
  `UpdateAuditOrgParams.Name` sends JSON `null`, which clears the stored name.
- `Audit.Orgs.Archive` / `Audit.Orgs.Unarchive` — idempotent org lifecycle
  (`POST /audit/orgs/{id}/archive`, `POST /audit/orgs/{id}/unarchive`).
  Archiving freezes new activity (ingest, streams, portal, and exports return
  `409 org_archived`); history stays verifiable.
- `Audit.Orgs.Delete` — hard delete (`DELETE /audit/orgs/{id}`), allowed only
  when nothing signed would be destroyed; otherwise the server returns
  `409 org_not_deletable`.
- `ListAuditOrgsParams.IncludeArchived` on `Audit.Orgs.List` — archived orgs
  are excluded by default; pass `IncludeArchived: true` to include them. Org
  objects now carry `archived_at` (RFC3339 string or null).
- `PATCH` support in the HTTP transport.
- `Client.Me` — key introspection (`GET /v1/me`), returning the raw decoded
  response (`map[string]any`): organization, tenant, API key metadata
  (prefix, last4, scopes), and effective limits. Requires no scopes.

### Changed

- **Breaking:** `Audit.Orgs.List` now takes a `ListAuditOrgsParams` struct;
  pass the zero value for the previous behavior.
- `Client.Validate` now probes `GET /v1/me` instead of `GET /v1/events?limit=1`.
  `/v1/me` requires no scopes, so keys limited to e.g. `audit:*` now validate
  correctly (the old events probe could 403 on scope and misreport). The
  `ValidationResult` contract is unchanged; only the 403 `Reason` text changed
  ("lacks permission to list events" → blocked by IP access rules, which is
  what a 403 from `/v1/me` means).

## [0.1.0] - 2026-07-06

### Added

- Initial release of the official Go SDK for the Invoance compliance API.
- `ComplianceEvent` reflects the real `GET /v1/events/:id` shape: `AccessTier` is optional (`*string`) and `ExpiresAt` is included.
- Synchronous, `context.Context`-aware client with functional options
  (`WithAPIKey`, `WithBaseURL`, `WithAPIVersion`, `WithTimeout`,
  `WithHTTPClient`, `WithIdempotencyKey`, `WithExtraHeaders`).
- Resources: `Events`, `Documents`, `Attestations`, `Traces`, and `Audit`
  (with `Events`, `Orgs`, `Streams`, `PortalSessions`, and `Exports`
  sub-resources).
- Single `*Error` type with an `ErrorKind` enum and `Is*` predicate helpers.
- Client-side crypto helpers: `invoance.audit/1` canonicalization
  (`CanonicalAuditBytes`, `PayloadHashHex`, `NormalizeTS`), offline Ed25519
  audit-event verification (`VerifyAuditEvent`), attestation signature
  verification (`Attestations.VerifySignature`), and
  `ContentIdempotencyKey`.
- `Client.Validate` health probe that never returns an error.
- Zero non-stdlib dependencies.
