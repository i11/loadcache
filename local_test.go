package loadcache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type testValue struct {
	value string
}

func TestLocalCacheNew(t *testing.T) {
	c := NewLocalCache[string, testValue]()
	assert.NotNil(t, c)
}

func TestLocalCacheGetEmpty(t *testing.T) {
	ctx := context.Background()
	c := NewLocalCache[string, *testValue]()
	assert.NotNil(t, c)
	v, err := c.Get(ctx, "bogus-123")
	assert.Nil(t, v)
	assert.NotNil(t, err)
	assert.ErrorIs(t, err, ErrNoCacheEntry)
}

func TestLocalCacheSet(t *testing.T) {
	ctx := context.Background()
	c := NewLocalCache[string, testValue]()
	assert.NotNil(t, c)
	id := "bogus-123"
	err := c.Set(ctx, id, testValue{value: "test"})
	assert.Nil(t, err)
	v, err := c.Get(ctx, id)
	assert.Nil(t, err)
	assert.NotNil(t, v)
	assert.Equal(t, "test", v.value)
}

func TestLocalCacheExpireWrite(t *testing.T) {
	ctx := context.Background()
	c := NewLocalCache[string, *testValue](
		WithExpireAfterWrite[string, *testValue](time.Second),
	)
	assert.NotNil(t, c)
	id := "bogus-123"
	err := c.Set(ctx, id, &testValue{value: "test"})
	assert.Nil(t, err)

	v, err := c.Get(ctx, id)
	assert.Nil(t, err)
	assert.NotNil(t, v)
	assert.Equal(t, "test", v.value)
	// Allow at least 2 ticks to pass
	time.Sleep(time.Second * 2)
	v, err = c.Get(ctx, id)
	assert.Nil(t, v)
	assert.NotNil(t, err)
	assert.ErrorIs(t, err, ErrNoCacheEntry)
}

func TestLocalCacheExpireAccess(t *testing.T) {
	ctx := context.Background()
	c := NewLocalCache[string, *testValue](WithExpireAfterAccess[string, *testValue](time.Second))
	assert.NotNil(t, c)
	id := "bogus-123"
	err := c.Set(ctx, id, &testValue{value: "test"})
	assert.Nil(t, err)
	v, err := c.Get(ctx, id)
	assert.Nil(t, err)
	assert.NotNil(t, v)
	time.Sleep(time.Millisecond * 500)
	v, err = c.Get(ctx, id)
	assert.Nil(t, err)
	assert.NotNil(t, v)
	time.Sleep(time.Millisecond * 500)
	v, err = c.Get(ctx, id)
	assert.Nil(t, err)
	assert.NotNil(t, v)
	// Allow at least 3 ticks to pass
	time.Sleep(time.Second * 3)
	v, err = c.Get(ctx, id)
	assert.Nil(t, v)
	assert.NotNil(t, err)
	assert.ErrorIs(t, err, ErrNoCacheEntry)
}

func TestLocalCacheSize(t *testing.T) {
	ctx := context.Background()
	c := NewLocalCache[string, testValue](WithCacheSize[string, testValue](1))
	assert.NotNil(t, c)
	err := c.Set(ctx, "bogus-123", testValue{value: "test"})
	assert.Nil(t, err)
	err = c.Set(ctx, "bogus-123", testValue{value: "test"})
	assert.Nil(t, err)
}

func TestLocalCacheDelete(t *testing.T) {
	ctx := context.Background()
	c := NewLocalCache[string, *testValue]()
	assert.NotNil(t, c)
	id := "bogus-123"
	err := c.Set(ctx, id, &testValue{value: "test"})
	assert.Nil(t, err)
	v, err := c.Get(ctx, id)
	assert.Nil(t, err)
	assert.NotNil(t, v)
	c.Delete(ctx, id)
	v, err = c.Get(ctx, id)
	assert.Nil(t, v)
	assert.NotNil(t, err)
	assert.ErrorIs(t, err, ErrNoCacheEntry)
}

func TestLocalCacheAsMap(t *testing.T) {
	ctx := context.Background()
	c := NewLocalCache[string, testValue]()
	assert.NotNil(t, c)
	id := "bogus-123"
	err := c.Set(ctx, id, testValue{value: "test"})
	assert.Nil(t, err)
	m := c.AsMap()
	assert.NotNil(t, m)
	assert.Len(t, m, 1)
	v, ok := m[id]
	assert.True(t, ok)
	assert.NotNil(t, v)
	assert.Equal(t, "test", v.value)
}

func TestLocalCacheOnEvict(t *testing.T) {
	ctx := context.Background()
	evicted := false
	c := NewLocalCache[string, testValue](
		WithExpireAfterWrite[string, testValue](time.Second),
		WithOnEvict[string, testValue](
			func(ctx context.Context, key string, value testValue) {
				evicted = true
			},
		))
	assert.NotNil(t, c)
	id := "bogus-123"
	err := c.Set(ctx, id, testValue{value: "test"})
	assert.Nil(t, err)
	// Allow at least 2 ticks to pass
	time.Sleep(time.Second * 2)
	assert.True(t, evicted)
}
