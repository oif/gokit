package cache

import (
	"errors"
	"math"
	"sync"
	"time"

	"github.com/cespare/xxhash"
)

// Errors
var (
	ErrNotExists = errors.New("cache object not exists")
)

const (
	bulkShard    = math.MaxUint8
	bulkAndValue = bulkShard - 1
)

// Bulk define interface which for bulk cache implementation(s)
type Bulk interface {
	Get(key string) (object interface{}, err error)
	Set(key string, object interface{}, ttl time.Duration) error
	TTL(key string) (ttl time.Duration, err error)
	Exist(key string) bool
	Delete(key string) error
}

type bucketDroplet struct {
	Payload   interface{}
	ExpiredAt time.Time
}

type bucketStore struct {
	droplets *sync.Map
}

func newBucket() *bucketStore {
	b := new(bucketStore)
	b.droplets = new(sync.Map)
	return b
}

// Returns never expire cache or non-expire object.
// If load a expired object, will execute lazy clean job and return not exists error as well.
func (b *bucketStore) getDroplet(key string) (bucketDroplet, error) {
	rawObject, ok := b.droplets.Load(key)
	if !ok {
		return bucketDroplet{}, ErrNotExists
	}
	droplet := rawObject.(bucketDroplet)
	if !droplet.ExpiredAt.IsZero() && droplet.ExpiredAt.Before(time.Now()) {
		b.droplets.Delete(key)
		return bucketDroplet{}, ErrNotExists
	}
	return droplet, nil
}

func (b *bucketStore) get(key string) (interface{}, error) {
	droplet, err := b.getDroplet(key)
	if err != nil {
		return nil, err
	}
	return droplet.Payload, nil
}
func (b *bucketStore) set(key string, object interface{}, ttl time.Duration) error {
	var expiredAt time.Time
	if ttl > 0 {
		expiredAt = time.Now().Add(ttl)
	}
	b.droplets.Store(key, bucketDroplet{
		Payload:   object,
		ExpiredAt: expiredAt,
	})
	return nil
}

func (b *bucketStore) ttl(key string) (time.Duration, error) {
	droplet, err := b.getDroplet(key)
	if err != nil {
		return 0, err
	}
	return time.Until(droplet.ExpiredAt), nil
}

func (b *bucketStore) exist(key string) bool {
	_, err := b.getDroplet(key)
	return err == nil
}

func (b *bucketStore) delete(key string) error {
	b.droplets.Delete(key)
	return nil
}

func hashFunc(key string) uint64 {
	return xxhash.Sum64String(key)
}

type bucket struct {
	buckets map[uint8]*bucketStore
}

// NewBulk return a sync map implement Bulk cache
func NewBulk() Bulk {
	b := new(bucket)
	b.buckets = make(map[uint8]*bucketStore, bulkShard)
	for i := uint8(0); i < bulkShard; i++ {
		b.buckets[i] = newBucket()
	}
	return b
}

func (b *bucket) getBucket(key string) *bucketStore {
	return b.buckets[uint8(hashFunc(key)&bulkAndValue)]
}

// Get object returns never expired(zero ttl) or non-expired content
func (b *bucket) Get(key string) (interface{}, error) {
	return b.getBucket(key).get(key)
}

// Set cache object with ttl, if set zero ttl means object will never expire
func (b *bucket) Set(key string, object interface{}, ttl time.Duration) error {
	return b.getBucket(key).set(key, object, ttl)
}

// TTL return time-to-live of object if exists
func (b *bucket) TTL(key string) (time.Duration, error) {
	return b.getBucket(key).ttl(key)
}

// Exist to check given object key is exists
func (b *bucket) Exist(key string) bool {
	return b.getBucket(key).exist(key)
}

// Delete cache by key manually
func (b *bucket) Delete(key string) error {
	return b.getBucket(key).delete(key)
}
