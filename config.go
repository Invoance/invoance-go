package invoance

import (
	"net/http"
	"os"
	"regexp"
	"time"
)

const (
	defaultBaseURL    = "https://api.invoance.com"
	defaultTimeout    = 30 * time.Second
	defaultAPIVersion = "v1"

	envAPIKey  = "INVOANCE_API_KEY"
	envBaseURL = "INVOANCE_BASE_URL"
)

var (
	trailingSlashes  = regexp.MustCompile(`/+$`)
	surroundingSlash = regexp.MustCompile(`^/+|/+$`)
)

// config is the resolved, immutable configuration used by the client.
type config struct {
	apiKey         string
	baseURL        string
	apiVersion     string
	timeout        time.Duration
	idempotencyKey string
	extraHeaders   map[string]string
	httpClient     *http.Client
}

// Option configures a Client. Options are applied in order by New.
type Option func(*config)

// WithAPIKey sets the API key explicitly, overriding INVOANCE_API_KEY.
func WithAPIKey(key string) Option {
	return func(c *config) { c.apiKey = key }
}

// WithBaseURL overrides the API host (default https://api.invoance.com, or
// the INVOANCE_BASE_URL environment variable). Trailing slashes are stripped.
func WithBaseURL(url string) Option {
	return func(c *config) { c.baseURL = url }
}

// WithAPIVersion sets the API version prefix (default "v1"). Surrounding
// slashes are stripped. The prefix is prepended to every request path.
func WithAPIVersion(version string) Option {
	return func(c *config) { c.apiVersion = version }
}

// WithTimeout sets the per-request timeout (default 30s).
func WithTimeout(d time.Duration) Option {
	return func(c *config) { c.timeout = d }
}

// WithHTTPClient supplies a custom *http.Client. When set, its Timeout is
// used as-is and WithTimeout is ignored for that client.
func WithHTTPClient(client *http.Client) Option {
	return func(c *config) { c.httpClient = client }
}

// WithIdempotencyKey sets a default Idempotency-Key sent with every mutating
// request. A per-call idempotency key overrides this default.
func WithIdempotencyKey(key string) Option {
	return func(c *config) { c.idempotencyKey = key }
}

// WithExtraHeaders merges additional headers into every request.
func WithExtraHeaders(headers map[string]string) Option {
	return func(c *config) {
		if c.extraHeaders == nil {
			c.extraHeaders = map[string]string{}
		}
		for k, v := range headers {
			c.extraHeaders[k] = v
		}
	}
}

// resolveConfig builds the effective config from options and environment.
func resolveConfig(opts ...Option) (config, error) {
	c := config{
		baseURL:      "",
		apiVersion:   "",
		timeout:      defaultTimeout,
		extraHeaders: map[string]string{},
	}
	for _, opt := range opts {
		opt(&c)
	}

	if c.apiKey == "" {
		c.apiKey = os.Getenv(envAPIKey)
	}
	if c.apiKey == "" {
		return config{}, &Error{
			Kind:    KindValidation,
			Message: "apiKey is required. Pass WithAPIKey(...) or set the " + envAPIKey + " environment variable.",
		}
	}

	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = os.Getenv(envBaseURL)
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	c.baseURL = trailingSlashes.ReplaceAllString(baseURL, "")

	apiVersion := c.apiVersion
	if apiVersion == "" {
		apiVersion = defaultAPIVersion
	}
	c.apiVersion = surroundingSlash.ReplaceAllString(apiVersion, "")

	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: c.timeout}
	}

	return c, nil
}
