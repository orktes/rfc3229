package util

import (
	"sync"
	"time"
)

type throttle struct {
	fn     func()
	cancel chan bool
}

type Throttler struct {
	sync.Mutex
	throttles    map[string]*throttle
	throttleTime time.Duration
}

func NewThrottler(throttleTime time.Duration) *Throttler {
	return &Throttler{
		throttles:    map[string]*throttle{},
		throttleTime: throttleTime,
	}
}

func (t *Throttler) Run(name string, fn func()) {
	go func() {
		closeCH := make(chan bool)

		t.Lock()
		if th, ok := t.throttles[name]; ok {
			close(th.cancel)
		}

		th := &throttle{
			fn,
			closeCH,
		}

		t.throttles[name] = th
		t.Unlock()

		select {
		case <-closeCH:
		case <-time.After(t.throttleTime):
			t.Lock()
			delete(t.throttles, name)
			t.Unlock()
			th.fn()
		}
	}()

}
