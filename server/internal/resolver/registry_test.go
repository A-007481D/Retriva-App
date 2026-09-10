package resolver

import (
	"context"
	"testing"
)

type mockResolver struct {
	canHandle bool
	info      *MediaInfo
	err       error
}

func (m *mockResolver) CanHandle(rawURL string) bool {
	return m.canHandle
}

func (m *mockResolver) Resolve(ctx context.Context, rawURL string) (*MediaInfo, error) {
	return m.info, m.err
}

func TestRegistry(t *testing.T) {
	info := &MediaInfo{URL: "test"}
	r1 := &mockResolver{canHandle: false}
	r2 := &mockResolver{canHandle: true, info: info}
	r3 := &mockResolver{canHandle: true} // should not be reached

	reg := NewRegistry(r1, r2, r3)

	got, err := reg.Resolve(context.Background(), "http://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != info {
		t.Errorf("expected info from r2, got %v", got)
	}
}

func TestRegistry_Unsupported(t *testing.T) {
	r1 := &mockResolver{canHandle: false}
	reg := NewRegistry(r1)

	_, err := reg.Resolve(context.Background(), "http://example.com")
	if err != ErrUnsupportedSource {
		t.Errorf("expected ErrUnsupportedSource, got %v", err)
	}
}
