package wait

import (
	"github.com/oif/gokit/runtime"
	"os"
	"os/signal"
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

// Execute f func and sleep period every time until stopCh is closed.
// If sliding is true, `f` first run will be executed after period. If it is false then
// `f` runs before time wait.
func Keep(f func(), period time.Duration, sliding bool, stopCh <-chan struct{}) {
	var (
		timer     = time.NewTimer(period)
		nextLoop  bool
		iteration = func() {
			defer func() {
				runtime.HandleCrash()
				if !timer.Stop() && !nextLoop {
					<-timer.C
				}
				timer.Reset(period)
			}()

			f()
		}
	)

	for {
		if !sliding {
			iteration()
		}

		select {
		case <-stopCh:
			return
		case <-timer.C:
			nextLoop = true
		}

		if sliding {
			iteration()
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

func Signal(signals ...os.Signal) {
	sig := make(chan os.Signal)
	signal.Notify(sig, signals...)
	Until(func() (bool, error) {
		get := <-sig
		for _, ts := range signals {
			if ts == get {
				return true, nil
			}
		}
		return false, nil
	})
}
