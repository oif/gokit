package main

import (
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"github.com/oif/gokit/leaderelection"
	"github.com/oif/gokit/wait"
	"github.com/thanhpk/randstr"
	"syscall"
	"time"
)

var (
	etcdClient *clientv3.Client
	instances  []*instance
)

type instance struct {
	group    string
	identity string
	le       *leaderelection.Elector
}

func (i *instance) onStarted(inFlight <-chan struct{}) {
	fmt.Printf("[%s/%s] started\n", i.group, i.identity)
}

func (i *instance) onStopped() {
	fmt.Printf("[%s/%s] stopped\n", i.group, i.identity)
}

func (i *instance) onEvent(e leaderelection.Event) {
	fmt.Printf("[%s/%s] ev: %v\n", i.group, i.identity, e)
}

func (i *instance) onNewLeader(identity string) {
	fmt.Printf("[%s/%s] new leader: %v\n", i.group, i.identity, identity)
}

func (i *instance) run() {
	le, err := leaderelection.New(leaderelection.Config{
		Group:         i.group,
		Identity:      i.identity,
		LeaseDuration: 3 * time.Second,
		RetryPeriod:   3 * time.Second,
		ETCDClient:    etcdClient,
		Callbacks: leaderelection.Callbacks{
			OnStartedLeading: i.onStarted,
			OnStoppedLeading: i.onStopped,
			OnEvent:          i.onEvent,
			OnNewLeader:      i.onNewLeader,
		},
	})
	if err != nil {
		panic(err)
	}
	go le.Run()
	i.le = le
	for {
		if le.IsLeader() {
			fmt.Printf("[%s/%s] Current LEADER\n", i.group, i.identity)
		}
		time.Sleep(3 * time.Second)
	}
}

func (i *instance) Shutdown() {
	i.le.Release()
}

func newElect() {
	i := &instance{
		group:    "example",
		identity: randstr.Hex(5),
	}
	go i.run()
	instances = append(instances, i)
}

func main() {
	var err error
	etcdClient, err = clientv3.New(clientv3.Config{
		Endpoints: []string{"127.0.0.1:2379"},
	})
	if err != nil {
		panic(err)
	}
	for i := 0; i < 2; i++ {
		newElect()
	}
	fmt.Println("All started")
	wait.Signal(syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	for _, i := range instances {
		i.Shutdown()
	}
}
