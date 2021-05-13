package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestInFlight(t *testing.T) {
	inFlight := NewInFlight()

	ginTest := func(req *http.Request) (*gin.Context, *gin.Engine, *httptest.ResponseRecorder) {
		w := httptest.NewRecorder()
		gin.SetMode(gin.ReleaseMode)
		ctx, engine := gin.CreateTestContext(w)
		ctx.Request = req
		return ctx, engine, w
	}

	testWait := func(blockDuration time.Duration) <-chan struct{} {
		req, err := http.NewRequest(http.MethodGet, "http://localhost/?t="+blockDuration.String(), nil)
		require.NoError(t, err)
		c, e, _ := ginTest(req)
		startCh := make(chan struct{})
		e.GET("/", inFlight.Track, func(c *gin.Context) {
			close(startCh)
			durationStr := c.Query("t")
			var duration time.Duration
			if durationStr != "" {
				duration, err = time.ParseDuration(durationStr)
				require.NoError(t, err)
			}
			time.Sleep(duration)
			c.Status(http.StatusOK)
		})
		go e.HandleContext(c)
		return startCh
	}

	blockDuration := time.Millisecond * 10

	<-testWait(0)
	require.NoError(t, inFlight.Wait(blockDuration))
	<-testWait(blockDuration)
	require.Equal(t, ErrInFlightWaitTimeout, inFlight.Wait(blockDuration/2))
	time.Sleep(blockDuration)
	require.NoError(t, inFlight.Wait(blockDuration))

	// non block
	testWait(blockDuration)
	blockDuration = time.Millisecond * 50
	<-testWait(blockDuration)
	require.Equal(t, ErrInFlightWaitTimeout, inFlight.Wait(blockDuration/2))
	require.NoError(t, inFlight.Wait(blockDuration))
}
