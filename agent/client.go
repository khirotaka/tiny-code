package agent

import (
	"context"

	"github.com/anthropics/anthropic-sdk-go"
	"golang.org/x/sync/semaphore"
)

// Client wraps the Anthropic SDK client and enforces a concurrency limit
// on simultaneous API calls across the entire agent tree.
type Client struct {
	inner *anthropic.Client
	sem   *semaphore.Weighted
}

// NewClient constructs a Client. concurrency limits simultaneous API calls.
func NewClient(concurrency int64) *Client {
	c := anthropic.NewClient()
	return &Client{
		inner: &c,
		sem:   semaphore.NewWeighted(concurrency),
	}
}

// call acquires a semaphore slot, delegates to the SDK, then releases the slot.
func (c *Client) call(ctx context.Context, params anthropic.MessageNewParams) (*anthropic.Message, error) {
	if err := c.sem.Acquire(ctx, 1); err != nil {
		return nil, err
	}
	defer c.sem.Release(1)
	return c.inner.Messages.New(ctx, params)
}
