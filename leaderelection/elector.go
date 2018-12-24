package leaderelection

import (
	"context"
	"errors"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/concurrency"
	"time"

	"github.com/oif/gokit/runtime"
	"github.com/oif/gokit/wait"
)

const (
	DefaultPrefix = "election"
)

var (
	ErrNonLeaderElected = errors.New("non-leader elected yet")
)

type Config struct {
	// LeaseDuration is the duration that non-leader candidates will
	// wait to force acquire leadership. This is measured against time of
	// last observed ack.
	LeaseDuration time.Duration
	// RetryPeriod is the duration the LeaderElector clients should wait
	// between tries of actions.
	RetryPeriod time.Duration

	// Callbacks are callbacks that are triggered during certain lifecycle
	// events of the LeaderElector
	Callbacks Callbacks

	// ETCDClient is used for connection with etcd cluster
	ETCDClient *clientv3.Client

	// Prefix is the prefix that used in etcd key construction
	Prefix string

	// Group is the group identify of instance group
	Group string
	// Identify is a unique id for current instance
	Identity string
}

// LeaderCallbacks are callbacks that are triggered during certain
// lifecycle events of the LeaderElector. These are invoked asynchronously.
//
// possible future callbacks:
//  * OnChallenge()
type Callbacks struct {
	// OnStartedLeading is called when a LeaderElector client starts leading
	OnStartedLeading func(inFlight <-chan struct{})
	// OnStoppedLeading is called when a LeaderElector client stops leading
	OnStoppedLeading func()
	// OnNewLeader is called when the client observes a leader that is
	// not the previously observed leader. This includes the first observed
	// leader when the client starts.
	OnNewLeader func(identity string)

	// OnEvent is called when a election events generated
	OnEvent func(e Event)
}

type Event struct {
	Group       string
	Identity    string
	RenewTime   time.Time
	AcquireTime time.Time
	Reason      string
}

type Elector struct {
	config        Config
	el            *concurrency.Election
	inFlight      chan struct{}
	currentLeader string
}

func New(c Config) (*Elector, error) {
	if c.Prefix == "" {
		c.Prefix = DefaultPrefix
	}
	el := new(Elector)
	el.config = c
	el.inFlight = make(chan struct{})
	return el, nil
}

func RunOrDie(lec Config) {
	le, err := New(lec)
	if err != nil {
		panic(err)
	}
	le.Run()
}

func (e *Elector) GetLeader() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), e.config.LeaseDuration)
	resp, err := e.el.Leader(ctx)
	if err != nil {
		cancel()
		return "", err
	}
	if len(resp.Kvs) == 0 {
		return "", ErrNonLeaderElected
	}
	return string(resp.Kvs[0].Value), nil
}

func (e *Elector) IsLeader() bool {
	return e.currentLeader == e.config.Identity
}

func (e *Elector) Run() {
	defer func() {
		runtime.HandleCrash()
	}()
	go e.config.Callbacks.OnStartedLeading(e.inFlight)
	e.elect()
}

func (e *Elector) Release() {
	close(e.inFlight)
	e.config.Callbacks.OnStoppedLeading()
	e.resign()
}

func (e *Elector) resign() {
	if e.el != nil {
		ctx, cancel := context.WithTimeout(context.Background(), e.config.LeaseDuration)
		err := e.el.Resign(ctx)
		if err != nil {
			cancel()
		}
	}
}

func (e *Elector) elect() {
	session, err := concurrency.NewSession(e.config.ETCDClient, concurrency.WithTTL(5))
	if err != nil {
		panic(err)
	}
	e.el = concurrency.NewElection(session, "/"+e.config.Prefix+"/"+e.config.Group)

	wait.Keep(func() {
		e.tryAcquireOrRenew()
		maybeNewLeader, err := e.GetLeader()
		if err == nil {
			if maybeNewLeader != e.currentLeader {
				e.config.Callbacks.OnNewLeader(maybeNewLeader)
				e.currentLeader = maybeNewLeader
			}
		}
	}, e.config.RetryPeriod, e.inFlight)
}

func (e *Elector) tryAcquireOrRenew() bool {
	now := time.Now()
	ev := Event{
		Group:       e.config.Group,
		Identity:    e.config.Identity,
		RenewTime:   now,
		AcquireTime: now,
	}
	ctx, cancel := context.WithTimeout(context.Background(), e.config.LeaseDuration)

	if err := e.el.Campaign(ctx, e.config.Identity); err != nil {
		cancel()
		// Acquire failed ignore this event
		if err != context.DeadlineExceeded {
			ev.Reason = err.Error()
			e.config.Callbacks.OnEvent(ev)
		}
		return false
	}
	return true
}
