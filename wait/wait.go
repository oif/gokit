package wait

import (
	"github.com/oif/gokit/runtime"
	"sync"
	"time"
)

type Group struct {
	wg sync.WaitGroup
}

func (g *Group) Wait() {
	g.wg.Wait()
}

func (g *Group) Run(f func()) {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		f()
	}()
}

// Execute f func and sleep period every time until stopCh is closed
func Keep(f func(), period time.Duration, stopCh chan struct{}) {
	var timer *time.Timer
	var timeout bool

	for {
		// Try if should stop
		select {
		case <-stopCh:
			return
		default:
		}

		timer = resetOrReuseTimer(timer, period, timeout)

		func() {
			defer runtime.HandleCrash()
			f()
		}()

		select {
		case <-stopCh:
			return
		case <-timer.C:
			timeout = true
		}
	}
}

type conditionFn func() (bool, error)

func Until(condition conditionFn) error {
	c := make(chan struct{})
	var err error
	go func(err error) {
		var (
			ok bool
		)
		for {
			ok, err = condition()
			if err != nil {
				close(c)
				return
			}
			if ok {
				close(c)
				return
			}
		}
	}(err)
	<-c
	return err
}

// resetOrReuseTimer avoids allocating a new timer if one is already in use.
// Not safe for multiple threads.
func resetOrReuseTimer(t *time.Timer, d time.Duration, timeout bool) *time.Timer {
	if t == nil {
		return time.NewTimer(d)
	}
	// If never stop and never timeout then wait timer expire
	if !t.Stop() && !timeout {
		<-t.C
	}
	t.Reset(d)
	return t
}
