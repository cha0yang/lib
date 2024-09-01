package ttlcache

import (
	"log"
	"time"
)

type hashable interface {
	string | uint64
}

type shard[K hashable, V any] struct {
	id int

	getMap func() map[K]V

	precision time.Duration

	index, maxIndex int
	timeWheel       []map[K]V
	mapping         map[K]int // key => index

	input chan input[K, V]

	expireFn func(map[K]V)
}

func newShard[K hashable, V any](id int, ttl, precision time.Duration, expireFn func(map[K]V), getMap func() map[K]V) *shard[K, V] {
	timeWheelPartCount := int(ttl / precision)

	c := &shard[K, V]{
		id:        id,
		getMap:    getMap,
		precision: precision,
		index:     0,
		maxIndex:  timeWheelPartCount - 1,
		timeWheel: make([]map[K]V, timeWheelPartCount),
		mapping:   map[K]int{},
		input:     make(chan input[K, V]),
		expireFn:  expireFn,
	}

	for i := 0; i < timeWheelPartCount; i++ {
		c.timeWheel[i] = getMap()
	}

	go c.process()

	return c
}

func (c *shard[K, V]) process() {
	ticker := time.NewTicker(c.precision)
	defer ticker.Stop()

	for {
		select {
		case input := <-c.input:
			bucketIdx, ok := c.mapping[input.k]
			if ok {
				if !input.updateTTL {
					continue
				}
				delete(c.mapping, input.k)
				delete(c.timeWheel[bucketIdx], input.k)
			}
			c.mapping[input.k] = c.index
			c.timeWheel[c.index][input.k] = input.v
		case <-ticker.C:
			expireIndex := c.index - 1
			if expireIndex == -1 {
				expireIndex = c.maxIndex
			}

			if expired := c.timeWheel[expireIndex]; len(expired) > 0 {
				c.timeWheel[expireIndex] = c.getMap()

				for k := range expired {
					delete(c.mapping, k)
				}

				c.expireFn(expired)

				log.Println("shard expired", time.Now(), c.id, len(expired))
			}

			if c.index++; c.index > c.maxIndex {
				c.index = 0
			}
		}
	}
}

func (c *shard[K, V]) set(k K, v V, updateTTL bool) {
	c.input <- input[K, V]{k, v, updateTTL}
}

type input[K hashable, V any] struct {
	k         K
	v         V
	updateTTL bool
}
