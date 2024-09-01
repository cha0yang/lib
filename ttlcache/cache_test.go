package ttlcache

import (
	"strconv"
	"testing"
	"time"
)

func Test1(t *testing.T) {
	hasher := func(k string) uint64 {
		return 0
	}

	count := 0
	expireFn := func(m map[string]int) {
		count += len(m)
	}

	c, err := NewCache[string, int](time.Second, time.Millisecond*100, hasher, expireFn)

	if err != nil {
		t.Log(err)
		return
	}

	go func() {
		for i := 0; i < 100; i++ {
			c.Set(strconv.Itoa(i), i, false)
			time.Sleep(time.Millisecond)
		}
	}()

	time.Sleep(time.Millisecond * 500)

	t.Log("======", count)
}

func Benchmark1(b *testing.B) {

}
