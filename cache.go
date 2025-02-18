package loadcache

import (
	"context"
)

type Cache[K comparable, V any] interface {
	Get(context.Context, K) (V, error)
	Set(context.Context, K, V) error
	Delete(context.Context, K)
	AsMap() map[K]V
	Cleanup(ctx context.Context)
}

type Loader[K comparable, V any] func(context.Context, K) (V, error)

type LoadingCache[K comparable, V any] interface {
	Cache[K, V]
	GetOrLoad(context.Context, K, ...Loader[K, V]) (V, error)
	Refresh(context.Context, K) error
}
