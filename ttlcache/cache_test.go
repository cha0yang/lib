package ttlcache

import (
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cespare/xxhash/v2"
)

var (
	generator = &atomic.Int64{}
	counter   = &atomic.Int64{}
	done      = &atomic.Bool{}
)

func set(c *Cache[string, int64], dur time.Duration) {
	defer done.Store(true)

	end := time.Now().Add(dur)
	for time.Now().Before(end) {
		n := generator.Add(1)
		c.Set(strconv.FormatInt(n, 10), n, false)
	}
}

func hasher(k string) uint64 {
	return xxhash.Sum64String(k)
}

func Test1(t *testing.T) {
	expireFn := func(m map[string]int64) {
		counter.Add(int64(len(m)))
	}

	c, err := NewCache[string, int64](time.Second*20, time.Millisecond*100, hasher, expireFn)
	if err != nil {
		t.Log(err)
		return
	}

	go set(c, time.Second*10)

	for range time.Tick(time.Second) {
		x1, x2 := generator.Load(), counter.Load()
		t.Log(time.Now(), x1, x2)

		if x1 == x2 && done.Load() {
			return
		}
	}
}
