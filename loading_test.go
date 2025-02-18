package loadcache

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var loader = func(_ context.Context, id int) (*testValue, error) {
	return &testValue{value: strconv.Itoa(id)}, nil
}

func TestLoadingCacheNew(t *testing.T) {
	c := NewLoadingCacheWithLoader[int, *testValue](loader)
	assert.NotNil(t, c)
}

func TestLoadingCacheGetLoad(t *testing.T) {
	ctx := context.Background()
	c := NewLoadingCacheWithLoader[int, *testValue](loader)
	assert.NotNil(t, c)
	v, err := c.Get(ctx, 1)
	assert.Nil(t, err)
	assert.NotNil(t, v)
	assert.Equal(t, "1", v.value)
}

func TestLoadingCacheRefresh(t *testing.T) {
	ctx := context.Background()
	c := NewLoadingCacheWithLoader[int, *testValue](loader, WithExpireAfterWrite[int, *testValue](time.Second))
	lc, ok := c.(*loadingCache[int, *testValue])
	assert.True(t, ok)
	assert.NotNil(t, c)
	v, err := c.Get(ctx, 1)
	assert.Nil(t, err)
	assert.NotNil(t, v)
	assert.Equal(t, "1", v.value)
	time.Sleep(time.Millisecond * 500)
	err = c.Refresh(ctx, 1)
	assert.Nil(t, err)
	// Allow at least 2 ticks to pass
	time.Sleep(time.Second * 2)
	v, err = lc.LocalCache.Get(ctx, 1)
	assert.Nil(t, v)
	assert.NotNil(t, err)
	assert.ErrorIs(t, err, ErrNoCacheEntry)
}
