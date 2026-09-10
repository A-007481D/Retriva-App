package resolver

import (
	"context"
	"errors"
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

// Resolve finds the first resolver that can handle the URL and calls it.
// Returns ErrUnsupportedSource if no resolver matches.
func (r *Registry) Resolve(ctx context.Context, rawURL string) (*MediaInfo, error) {
	for _, res := range r.resolvers {
		if res.CanHandle(rawURL) {
			return res.Resolve(ctx, rawURL)
		}
	}
	return nil, ErrUnsupportedSource
}
