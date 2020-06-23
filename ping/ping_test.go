package ping_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/oif/gokit/ping"
)

func TestPing(t *testing.T) {
	RTs, err := ping.Ping("meitu.com", 10, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	for _, RT := range RTs {
		fmt.Println(RT)
	}
}
