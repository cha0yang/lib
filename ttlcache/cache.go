package ttlcache

import (
	"errors"
	"runtime"
	"sync"
	"time"

	_ "go.uber.org/automaxprocs"
	"golang.org/x/exp/maps"
)

type Cache[K hashable, V any] struct {
	hasher func(K) uint64

	shardCount int
	shards     []*shard[K, V]

	expireFn      func(map[K]V)
	expireChannel chan map[K]V

	mapPool *sync.Pool
}

func NewCache[K hashable, V any](ttl, precision time.Duration, hasher func(K) uint64, expireFn func(map[K]V)) (*Cache[K, V], error) {
	if ttl < time.Millisecond || precision < time.Millisecond || ttl/precision <= 1 || ttl%precision > 0 {
		return nil, errors.New("ttl must >= 1ms && precision must >= 1ms && ttl/precision must > 1 && ttl must be divisible by precision")
	}

	shardCount := runtime.NumCPU()

	c := &Cache[K, V]{
		hasher:        hasher,
		shardCount:    shardCount,
		shards:        make([]*shard[K, V], shardCount),
		expireFn:      expireFn,
		expireChannel: make(chan map[K]V, shardCount),
		mapPool:       &sync.Pool{New: func() any { return map[K]V{} }},
	}

	for i := 0; i < shardCount; i++ {
		c.shards[i] = newShard(i, ttl, precision, c.expireFnForSharding, c.getMap)
	}

	go c.process()

	return c, nil
}

func (c *Cache[K, V]) getMap() map[K]V {
	return c.mapPool.Get().(map[K]V)
}

func (c *Cache[K, V]) expireFnForSharding(expired map[K]V) {
	c.expireChannel <- expired
}

func (c *Cache[K, V]) process() {
	for expired := range c.expireChannel {
		c.expireFn(expired)

		maps.Clear(expired)
		c.mapPool.Put(expired)
	}
}

func (c *Cache[K, V]) Set(k K, v V, updateTTL bool) {
	index := int(c.hasher(k)) % c.shardCount
	if index < 0 {
		index = -index
	}

	c.shards[index].set(k, v, updateTTL)
}
