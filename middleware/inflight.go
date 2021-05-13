package middleware

import (
	"errors"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type InFlight struct {
	wg sync.WaitGroup
}

func NewInFlight() *InFlight {
	return new(InFlight)
}

var ErrInFlightWaitTimeout = errors.New("in flight request wait timeout")

func (i *InFlight) Track(c *gin.Context) {
	i.wg.Add(1)
	c.Next()
	i.wg.Done()
}

func (i *InFlight) Wait(timeout time.Duration) error {
	completed := make(chan struct{})
	go func() {
		i.wg.Wait()
		close(completed)
	}()
	timer := time.NewTimer(timeout)
	select {
	case <-timer.C:
		return ErrInFlightWaitTimeout
	case <-completed:
		return nil
	}
}
