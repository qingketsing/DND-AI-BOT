package worker

import (
	"context"
	"time"
)

type fakeLockClient struct {
	values map[string]string
}

func newFakeLockClient() *fakeLockClient {
	return &fakeLockClient{values: make(map[string]string)}
}

func (c *fakeLockClient) SetNX(ctx context.Context, key string, value string, ttl time.Duration) (bool, error) {
	if _, ok := c.values[key]; ok {
		return false, nil
	}
	c.values[key] = value
	return true, nil
}

func (c *fakeLockClient) CompareAndExpire(ctx context.Context, key string, expected string, ttl time.Duration) (bool, error) {
	if value, ok := c.values[key]; ok && value == expected {
		return true, nil
	}
	return false, nil
}

func (c *fakeLockClient) CompareAndDelete(ctx context.Context, key string, expected string) (bool, error) {
	if value, ok := c.values[key]; ok && value == expected {
		delete(c.values, key)
		return true, nil
	}
	return false, nil
}
