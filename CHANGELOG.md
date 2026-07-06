# Changelog

All notable changes to this project are documented here. The format is based
on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
