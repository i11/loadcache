package loadcache

import (
	"context"
	"errors"
	"sync"
	"time"
)

const (
	defaultSize         = 100
	defaultTickInterval = time.Second
)

var ErrNoCacheEntry = errors.New("no cache entry")

type record[T any] struct {
	entry    T
	accessAt time.Time
	writeAt  time.Time
}

type LocalCacheOption[K comparable, V any] func(*LocalCache[K, V])

// LocalCache is a simple and rather naive in-memory cache implementation,
// exist only for code and module dependency simplicity.
// Having a single lock for the whole storage is one of the main bottlenecks for this implementation.
// To reduce copying of cached values, pointers are used. However, this doesn't offer proper access control
// and allow mutation of cached values from outside the cache implementation.
// It doesn't implement any of the eviction strategies (e.g. LRU, LFU, ARC).
// Instead, it relies on map overwriting existing entries if there is no room for new ones.
// There is no scaling or sharding support either. Cache is maintained by a single goroutine.
// For large caches, it is recommended to use a more sophisticated implementation such as
// bigcache, ristretto, fastcache, gocache and similar.
type LocalCache[K comparable, V any] struct {
	stop chan struct{}

	wg           sync.WaitGroup
	mu           sync.RWMutex
	store        map[K]*record[V]
	accessExpire *time.Duration
	writeExpire  *time.Duration
	tickInterval time.Duration
	onEvict      func(context.Context, K, V)
}

var _ Cache[int, any] = (*LocalCache[int, any])(nil)

func (c *LocalCache[K, V]) unsafeEvict(ctx context.Context, id K, record *record[V]) {
	if c.onEvict != nil {
		c.onEvict(ctx, id, record.entry)
	}
	delete(c.store, id)
}

func (c *LocalCache[K, V]) unsafeAccessExpire(ctx context.Context, id K, rec *record[V], now time.Time) bool {
	if c.accessExpire != nil && rec.accessAt.Add(*c.accessExpire).Before(now) {
		c.unsafeEvict(ctx, id, rec)
		return true
	}
	return false
}

func (c *LocalCache[K, V]) unsafeWriteExpire(ctx context.Context, id K, rec *record[V], now time.Time) bool {
	if c.writeExpire != nil && rec.writeAt.Add(*c.writeExpire).Before(now) {
		c.unsafeEvict(ctx, id, rec)
		return true
	}
	return false
}

func (c *LocalCache[K, V]) cleanup(ctx context.Context, tick time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for id, rec := range c.store {
		evicted := c.unsafeAccessExpire(ctx, id, rec, tick)
		if !evicted {
			c.unsafeWriteExpire(ctx, id, rec, tick)
		}
	}
}

func (c *LocalCache[K, V]) cleanupLoop(ctx context.Context) {
	t := time.NewTicker(c.tickInterval)
	defer t.Stop()

	for {
		select {
		case <-c.stop:
			return
		case tick := <-t.C:
			c.cleanup(ctx, tick)
		}
	}
}

func (c *LocalCache[K, V]) Get(_ context.Context, id K) (entry V, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	record, ok := c.store[id]
	if ok {
		entry = record.entry
		record.accessAt = time.Now()
	} else {
		err = ErrNoCacheEntry
	}

	return entry, err
}

func (c *LocalCache[K, V]) Set(_ context.Context, id K, entry V) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	c.store[id] = &record[V]{
		entry:    entry,
		accessAt: now,
		writeAt:  now,
	}
	return nil
}

func (c *LocalCache[K, V]) Delete(ctx context.Context, id K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.unsafeEvict(ctx, id, c.store[id])
}

func (c *LocalCache[K, V]) Cleanup(ctx context.Context) {
	c.cleanup(ctx, time.Now())
}

func (c *LocalCache[K, V]) AsMap() map[K]V {
	c.mu.RLock()
	defer c.mu.RUnlock()

	m := make(map[K]V, len(c.store))
	for uid, cu := range c.store {
		m[uid] = cu.entry
	}
	return m
}

func (c *LocalCache[K, V]) Start(ctx context.Context) {
	c.wg.Add(1)
	go func(ctx context.Context) {
		defer c.wg.Done()
		c.cleanupLoop(ctx)
	}(ctx)
}

func (c *LocalCache[K, V]) Stop() {
	close(c.stop)
	c.wg.Wait()
}

func NewLocalCache[K comparable, V any](options ...LocalCacheOption[K, V]) Cache[K, V] {
	lc := &LocalCache[K, V]{
		tickInterval: defaultTickInterval,
		store:        make(map[K]*record[V], defaultSize),
		stop:         make(chan struct{}),
	}
	for _, option := range options {
		option(lc)
	}
	ctx := context.Background()
	lc.Start(ctx)
	return lc
}

func WithCacheSize[K comparable, V any](size int) LocalCacheOption[K, V] {
	return func(c *LocalCache[K, V]) {
		c.store = make(map[K]*record[V], size)
	}
}

func WithExpireAfterAccess[K comparable, V any](expire time.Duration) LocalCacheOption[K, V] {
	return func(c *LocalCache[K, V]) {
		c.accessExpire = &expire
	}
}

func WithExpireAfterWrite[K comparable, V any](expire time.Duration) LocalCacheOption[K, V] {
	return func(c *LocalCache[K, V]) {
		c.writeExpire = &expire
	}
}

func WithTickInternal[K comparable, V any](interval time.Duration) LocalCacheOption[K, V] {
	return func(c *LocalCache[K, V]) {
		c.tickInterval = interval
	}
}

func WithOnEvict[K comparable, V any](onEvict func(context.Context, K, V)) LocalCacheOption[K, V] {
	return func(c *LocalCache[K, V]) {
		c.onEvict = onEvict
	}
}
