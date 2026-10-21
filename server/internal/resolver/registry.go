package resolver

import (
	"context"
	"errors"
	"strings"
)

// ErrUnsupportedSource is returned when no resolver can handle the given URL.
var ErrUnsupportedSource = errors.New("unsupported media source")

// Registry maintains an ordered list of resolvers and dispatches URLs to the first one that can handle it.
type Registry struct {
	resolvers []Resolver
}

// NewRegistry creates a new Registry with the given resolvers.
// Order matters: resolvers are checked in the order they are provided.
func NewRegistry(resolvers ...Resolver) *Registry {
	return &Registry{
		resolvers: append([]Resolver(nil), resolvers...),
	}
}

// Resolve finds resolvers that can handle the URL and calls them in order.
// If a resolver fails, it attempts subsequent candidate resolvers before returning the error.
func (r *Registry) Resolve(ctx context.Context, rawURL string) (*MediaInfo, error) {
	var lastErr error
	for _, res := range r.resolvers {
		if res.CanHandle(rawURL) {
			info, err := res.Resolve(ctx, rawURL)
			if err == nil {
				return info, nil
			}
			lastErr = err
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, ErrUnsupportedSource
}

// Inspect finds resolvers implementing Inspector that can handle the URL.
// If an inspector fails, it attempts subsequent candidate inspectors before returning the error.
func (r *Registry) Inspect(ctx context.Context, rawURL string) (*PlaylistInfo, error) {
	var lastErr error
	for _, res := range r.resolvers {
		if res.CanHandle(rawURL) {
			if insp, ok := res.(Inspector); ok {
				info, err := insp.Inspect(ctx, rawURL)
				if err == nil {
					return info, nil
				}
				lastErr = err
				if strings.Contains(err.Error(), "no active stories") {
					return nil, err
				}
			}
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, ErrUnsupportedSource
}
