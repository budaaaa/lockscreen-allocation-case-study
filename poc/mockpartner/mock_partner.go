// Package mockpartner provides local partner behavior for the proof of concept.
package mockpartner

import (
	"context"
	"errors"
	"time"

	"case-study-poc/allocation"
)

// MockPartner returns configured items after a delay or a configured failure.
type MockPartner struct {
	PartnerName string
	Delay       time.Duration
	Fail        bool
	Items       []allocation.Item
}

func (p MockPartner) Name() string { return p.PartnerName }

func (p MockPartner) Fetch(ctx context.Context, n int) ([]allocation.Item, error) {
	if p.Delay > 0 {
		timer := time.NewTimer(p.Delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if p.Fail {
		return nil, errors.New("mock partner unavailable")
	}
	if n < len(p.Items) {
		return append([]allocation.Item(nil), p.Items[:n]...), nil
	}
	return append([]allocation.Item(nil), p.Items...), nil
}
