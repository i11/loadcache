package loadcache

import (
	"context"
	"time"
)

type loadingCache[K comparable, V any] struct {
	*LocalCache[K, V]
	load Loader[K, V]
}

var _ LoadingCache[int, any] = (*loadingCache[int, any])(nil)

func (c *loadingCache[K, V]) GetOrLoad(ctx context.Context, id K, loaders ...Loader[K, V]) (entry V, err error) {
	c.mu.RLock()

	record, ok := c.store[id]
	if ok {
		entry = record.entry
		record.accessAt = time.Now()
		c.mu.RUnlock()
	} else {
		c.mu.RUnlock()
		for _, load := range loaders {
			if entry, err = load(ctx, id); err == nil {
				err = c.Set(ctx, id, entry)
				return entry, err
			}
		}
	}
	return entry, err
}

func (c *loadingCache[K, V]) Get(ctx context.Context, id K) (entry V, err error) {
	c.mu.RLock()

	record, ok := c.store[id]
	switch ok {
	case true:
		entry = record.entry
		record.accessAt = time.Now()
		c.mu.RUnlock()
	case false:
		c.mu.RUnlock()
		if c.load != nil {
			if entry, err = c.load(ctx, id); err == nil {
				err = c.Set(ctx, id, entry)
			}
		} else {
			err = ErrNoCacheEntry
		}
	}

	return entry, err
}

func (c *loadingCache[K, V]) Refresh(ctx context.Context, id K) (err error) {
	c.mu.RLock()
	_, ok := c.store[id]
	c.mu.RUnlock()
	if ok {
		var entry V
		if entry, err = c.load(ctx, id); err == nil {
			err = c.Set(ctx, id, entry)
		}
	} else {
		err = ErrNoCacheEntry
	}
	return err
}

func NewLoadingCache[K comparable, V any](opts ...LocalCacheOption[K, V]) LoadingCache[K, V] {
	return &loadingCache[K, V]{
		LocalCache: NewLocalCache(opts...).(*LocalCache[K, V]),
	}
}

func NewLoadingCacheWithLoader[K comparable, V any](
	load Loader[K, V],
	opts ...LocalCacheOption[K, V],
) LoadingCache[K, V] {
	return &loadingCache[K, V]{
		LocalCache: NewLocalCache(opts...).(*LocalCache[K, V]),
		load:       load,
	}
}
