package resolver

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LoaderFunc retrieves a requested release through an ID format or a mapping.
// The resolver verifies that the returned metadata matches both fields.
type LoaderFunc func(reference ExtensionReference) (io.ReadCloser, error)

func NewHTTPLoader(timeout time.Duration) LoaderFunc {
	client := &http.Client{Timeout: timeout}
	return func(reference ExtensionReference) (io.ReadCloser, error) {
		resp, err := client.Get(reference.ID)
		if err != nil {
			return nil, fmt.Errorf("fetching %s: %w", reference, err)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("fetching %s: unexpected status %s", reference, resp.Status)
		}
		return resp.Body, nil
	}
}

func NewInMemoryLoader(docs map[ExtensionReference][]byte) LoaderFunc {
	return func(reference ExtensionReference) (io.ReadCloser, error) {
		data, ok := docs[reference]
		if !ok {
			return nil, &FetchError{Reference: reference, Err: ErrNotFound}
		}
		return io.NopCloser(bytes.NewReader(data)), nil
	}
}
