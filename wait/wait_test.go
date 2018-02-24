package wait_test

import (
	"math"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/oif/gokit/wait"
)

var (
	targetSig = []os.Signal{syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM}
)

func TestUntil(t *testing.T) {
	sig := make(chan os.Signal)
	signal.Notify(sig, targetSig...)

	go func() {
		t.Log("Send SIGQUIT")
		sig <- syscall.SIGQUIT
		t.Log("Wait 1s to send SIGHUP")
		time.Sleep(500 * time.Millisecond)
		t.Log("Send SIGHUP")
		sig <- syscall.SIGHUP
	}()
	t.Log("Waiting")
	wait.Until(func() (bool, error) {
		t.Log("[Until] waiting")
		get := <-sig
		t.Logf("[Until] get %s", get)
		for _, ts := range targetSig {
			if ts == get {
				return true, nil
			}
		}
		return false, nil
	})
	t.Log("Stop waiting")
}

func TestKeep(t *testing.T) {
	stop := make(chan struct{})
	var (
		base  = 1
		times = 10
	)

	go func() {
		time.Sleep(time.Duration(times) * time.Millisecond)
		close(stop)
	}()

	wait.Keep(func() {
		base++
	}, time.Millisecond, stop)
	if math.Abs(float64(base-times)) > 1 {
		t.Fatalf("expect %d got %d", times, base)
	}
}
