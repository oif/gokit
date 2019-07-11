// +build !windows

package dumpstack

import (
	"os"
	"os/signal"
	"syscall"
)

func SetupTrap() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGUSR1)
	go func() {
		for range c {
			PrintStack()
		}
	}()
}
