package dynamictools

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/utils"
)

// maxHTTPFetchRedirects limits redirect chains for schema-driven API fetches.
const maxHTTPFetchRedirects = 8

// newFetchHTTPClient returns an HTTP client with redirect validation and SSRF-safe defaults.
func newFetchHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxHTTPFetchRedirects {
				return fmt.Errorf("too many redirects")
			}
			if err := validateFetchURL(req.Context(), req.URL.String()); err != nil {
				return err
			}
			return nil
		},
	}
}

// validateFetchURL delegates to utils.ValidateURLForRequest for SSRF protection.
func validateFetchURL(ctx context.Context, raw string) error {
	_, err := utils.ValidateURLForRequest(ctx, raw)
	return err
}
