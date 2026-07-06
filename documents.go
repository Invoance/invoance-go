package invoance

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"os"
	"path/filepath"
)

// DocumentsResource is the document-anchoring resource (client.Documents).
type DocumentsResource struct {
	t *transport
}

// AnchorDocumentParams are the parameters for Documents.Anchor.
type AnchorDocumentParams struct {
	// DocumentHash is the required 64-char lowercase hex SHA-256 digest.
	DocumentHash string
	// DocumentRef is an optional human-readable reference.
	DocumentRef string
	// EventType is an optional classification string.
	EventType string
	// OriginalBytesB64 optionally uploads the original bytes (base64).
	OriginalBytesB64 string
	// Metadata is optional arbitrary JSON metadata.
	Metadata map[string]any
	// IdempotencyKey is an optional per-call idempotency key.
	IdempotencyKey string
	// TraceID optionally associates the document with a trace.
	TraceID string
}

// Anchor anchors a document hash (POST /document/anchor).
func (r *DocumentsResource) Anchor(ctx context.Context, params AnchorDocumentParams) (AnchorDocumentResponse, error) {
	var out AnchorDocumentResponse
	if err := assertSha256Hex("documentHash", params.DocumentHash); err != nil {
		return out, err
	}
	body := map[string]any{
		"document_hash": params.DocumentHash,
	}
	if params.DocumentRef != "" {
		body["document_ref"] = params.DocumentRef
	}
	if params.EventType != "" {
		body["event_type"] = params.EventType
	}
	if params.OriginalBytesB64 != "" {
		body["original_bytes_b64"] = params.OriginalBytesB64
	}
	if params.Metadata != nil {
		body["metadata"] = params.Metadata
	}
	if params.TraceID != "" {
		body["trace_id"] = params.TraceID
	}
	err := r.t.post(ctx, "/document/anchor", body, params.IdempotencyKey, &out)
	return out, err
}

// AnchorFileParams are the parameters for Documents.AnchorFile. Provide
// exactly one of Path or Bytes.
type AnchorFileParams struct {
	// Path is a file path on disk. DocumentRef defaults to its basename.
	Path string
	// Bytes is the file content as raw bytes.
	Bytes []byte
	// DocumentRef is an optional human-readable reference.
	DocumentRef string
	// EventType is an optional classification string.
	EventType string
	// Metadata is optional arbitrary JSON metadata.
	Metadata map[string]any
	// IdempotencyKey is an optional per-call idempotency key.
	IdempotencyKey string
	// SkipOriginal skips uploading the original file bytes when true.
	SkipOriginal bool
	// TraceID optionally associates the document with a trace.
	TraceID string
}

// AnchorFile reads a file (Path or Bytes), computes its SHA-256 hash,
// base64-encodes the bytes (unless SkipOriginal), and calls Anchor.
func (r *DocumentsResource) AnchorFile(ctx context.Context, params AnchorFileParams) (AnchorDocumentResponse, error) {
	var out AnchorDocumentResponse
	content := params.Bytes
	if params.Path != "" {
		data, err := os.ReadFile(params.Path)
		if err != nil {
			return out, &Error{Kind: KindValidation, Message: "failed to read file: " + err.Error(), cause: err}
		}
		content = data
	}

	sum := sha256.Sum256(content)
	documentHash := hex.EncodeToString(sum[:])

	documentRef := params.DocumentRef
	if documentRef == "" && params.Path != "" {
		documentRef = filepath.Base(params.Path)
	}

	anchor := AnchorDocumentParams{
		DocumentHash:   documentHash,
		DocumentRef:    documentRef,
		EventType:      params.EventType,
		Metadata:       params.Metadata,
		IdempotencyKey: params.IdempotencyKey,
		TraceID:        params.TraceID,
	}
	if !params.SkipOriginal {
		anchor.OriginalBytesB64 = base64.StdEncoding.EncodeToString(content)
	}
	return r.Anchor(ctx, anchor)
}

// ListDocumentsParams are the query parameters for Documents.List.
type ListDocumentsParams struct {
	Page        *int
	Limit       *int
	DateFrom    string
	DateTo      string
	DocumentRef string
}

// List returns a paginated document listing (GET /document).
func (r *DocumentsResource) List(ctx context.Context, params ListDocumentsParams) (ListDocumentsResponse, error) {
	var out ListDocumentsResponse
	err := r.t.get(ctx, "/document", map[string]any{
		"page":         params.Page,
		"limit":        params.Limit,
		"date_from":    params.DateFrom,
		"date_to":      params.DateTo,
		"document_ref": params.DocumentRef,
	}, &out)
	return out, err
}

// Get retrieves a single document (GET /document/{eventID}).
func (r *DocumentsResource) Get(ctx context.Context, eventID string) (DocumentEvent, error) {
	var out DocumentEvent
	err := r.t.get(ctx, "/document/"+eventID, nil, &out)
	return out, err
}

// GetOriginal downloads the original document bytes
// (GET /document/{eventID}/original).
func (r *DocumentsResource) GetOriginal(ctx context.Context, eventID string) ([]byte, error) {
	return r.t.getBytes(ctx, "/document/"+eventID+"/original")
}

// VerifyDocumentParams are the parameters for Documents.Verify.
type VerifyDocumentParams struct {
	// DocumentHash is the 64-char lowercase hex SHA-256 digest to check.
	DocumentHash string
}

// Verify checks a submitted hash against the anchored document
// (POST /document/{eventID}/verify).
func (r *DocumentsResource) Verify(ctx context.Context, eventID string, params VerifyDocumentParams) (VerifyDocumentResponse, error) {
	var out VerifyDocumentResponse
	if err := assertSha256Hex("documentHash", params.DocumentHash); err != nil {
		return out, err
	}
	err := r.t.post(ctx, "/document/"+eventID+"/verify", map[string]any{
		"document_hash": params.DocumentHash,
	}, "", &out)
	return out, err
}
