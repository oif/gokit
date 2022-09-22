//go:generate protoc -I . -I $GOPATH/src --go_out=. loki.proto
// nolint
package loki

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/golang/snappy"
	"github.com/prometheus/common/model"
)

const (
	contentType  = "application/x-protobuf"
	postPath     = "/api/prom/push"
	maxErrMsgLen = 1024
)

type entry struct {
	labels model.LabelSet
	*EntryAdapter
}

type payload struct {
	at     time.Time
	labels map[string]string
	line   string
}

type Loki struct {
	entry
	LokiURL       string
	BatchWait     time.Duration
	BatchSize     int
	payloadCh     chan payload
	hostname      string
	prependLabels map[model.LabelName]model.LabelValue
	wg            sync.WaitGroup
	username      string
	password      string
	customHeader  map[string]string
}

func NewLoki(URL string, batchSize, batchWait int, username, password string, customHeader map[string]string) (*Loki, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	return NewLokiCustomHostname(URL, batchSize, batchWait, hostname, username, password, customHeader)
}

func NewLokiCustomHostname(URL string, batchSize, batchWait int, hostname, username, password string, customHeader map[string]string) (*Loki, error) {
	l := &Loki{
		LokiURL:       URL,
		BatchSize:     batchSize,
		BatchWait:     time.Duration(batchWait) * time.Second,
		payloadCh:     make(chan payload, batchSize),
		prependLabels: make(map[model.LabelName]model.LabelValue),
		hostname:      hostname,
		username:      username,
		password:      password,
		customHeader:  customHeader,
	}

	u, err := url.Parse(l.LokiURL)
	if err != nil {
		return nil, err
	}
	if !strings.Contains(u.Path, postPath) {
		u.Path = postPath
		q := u.Query()
		u.RawQuery = q.Encode()
		l.LokiURL = u.String()
	}
	l.wg.Add(1)
	go l.run()
	return l, nil
}

func (l *Loki) Close() {
	close(l.payloadCh)
	l.wg.Wait()
}

func (l *Loki) AddPrependLabel(key, value string) {
	l.prependLabels[model.LabelName(key)] = model.LabelValue(value)
}

func (l *Loki) Send(at time.Time, labels map[string]string, line string) {
	l.payloadCh <- payload{
		at:     at,
		labels: labels,
		line:   line,
	}
}

func (l *Loki) run() {
	var (
		curPktTime  time.Time
		lastPktTime time.Time
		maxWait     = time.NewTimer(l.BatchWait)
		batch       = map[model.Fingerprint]*StreamAdapter{}
		batchSize   = 0
	)
	defer l.wg.Done()

	defer func() {
		if err := l.sendBatch(batch); err != nil {
			fmt.Fprintf(os.Stderr, "%v ERROR: loki flush: %v\n", time.Now(), err)
		}
	}()

	for {
		select {
		case p, ok := <-l.payloadCh:
			if !ok {
				return
			}
			curPktTime = p.at
			// guard against entry out of order errors
			if lastPktTime.After(curPktTime) {
				curPktTime = time.Now()
			}
			lastPktTime = curPktTime

			tsNano := curPktTime.UnixNano()
			ts := &timestamp.Timestamp{
				Seconds: tsNano / int64(time.Second),
				Nanos:   int32(tsNano % int64(time.Second)),
			}

			l.entry = entry{model.LabelSet{}, &EntryAdapter{Timestamp: ts}}
			for key, value := range p.labels {
				l.entry.labels[model.LabelName(key)] = model.LabelValue(value)
			}
			for key, value := range l.prependLabels {
				l.entry.labels[key] = value
			}
			l.entry.EntryAdapter.Line = p.line

			if batchSize+len(l.entry.Line) > l.BatchSize {
				if err := l.sendBatch(batch); err != nil {
					fmt.Fprintf(os.Stderr, "%v ERROR: send size batch: %v\n", lastPktTime, err)
				}
				batchSize = 0
				batch = map[model.Fingerprint]*StreamAdapter{}
				maxWait.Reset(l.BatchWait)
			}

			batchSize += len(l.entry.Line)
			fp := l.entry.labels.FastFingerprint()
			stream, ok := batch[fp]
			if !ok {
				stream = &StreamAdapter{
					Labels: l.entry.labels.String(),
				}
				batch[fp] = stream
			}
			stream.Entries = append(stream.Entries, l.EntryAdapter)

		case <-maxWait.C:
			if len(batch) > 0 {
				if err := l.sendBatch(batch); err != nil {
					fmt.Fprintf(os.Stderr, "%v ERROR: send time batch: %v\n", lastPktTime, err)
				}
				batchSize = 0
				batch = map[model.Fingerprint]*StreamAdapter{}
			}
			maxWait.Reset(l.BatchWait)
		}
	}
}

func (l *Loki) sendBatch(batch map[model.Fingerprint]*StreamAdapter) error {
	buf, err := encodeBatch(batch)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = l.send(ctx, buf)
	if err != nil {
		return err
	}
	return nil
}

func encodeBatch(batch map[model.Fingerprint]*StreamAdapter) ([]byte, error) {
	req := PushRequest{
		Streams: make([]*StreamAdapter, 0, len(batch)),
	}
	for _, stream := range batch {
		req.Streams = append(req.Streams, stream)
	}
	buf, err := proto.Marshal(&req)
	if err != nil {
		return nil, err
	}
	buf = snappy.Encode(nil, buf)
	return buf, nil
}

func (l *Loki) send(ctx context.Context, buf []byte) (int, error) {
	req, err := http.NewRequest("POST", l.LokiURL, bytes.NewReader(buf))
	if err != nil {
		return -1, err
	}
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", contentType)
	if l.password != "" {
		req.SetBasicAuth(l.username, l.password)
	}
	for key, value := range l.customHeader {
		req.Header.Set(key, value)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		scanner := bufio.NewScanner(io.LimitReader(resp.Body, maxErrMsgLen))
		line := ""
		if scanner.Scan() {
			line = scanner.Text()
		}
		err = fmt.Errorf("server returned HTTP status %s (%d): %s", resp.Status, resp.StatusCode, line)
	}
	return resp.StatusCode, err
}
