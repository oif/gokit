// nolint
//
//go:generate protoc -I . -I $GOPATH/src --go_out=. loki.proto
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
	postPath     = "/loki/api/v1/push"
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
	lokiURL         string
	batchWait       time.Duration
	batchSize       int
	payloadCh       chan payload
	hostname        string
	prependLabels   map[model.LabelName]model.LabelValue
	wg              sync.WaitGroup
	username        string
	password        string
	customHeader    map[string]string
	lokiClient      *http.Client
	honorOriginTime bool
}

type Options struct {
	BatchSize       int // send message lines in one streams
	BatchWait       int
	Username        string
	Password        string
	LokiTimeout     int  // always make sure calling loki api can be timed out
	HonorOriginTime bool // keep the message time as is rather than changing to current even though messages lagging behind
}

func WithBatch(batchSize, batchWait int) func(*Options) {
	return func(o *Options) {
		o.BatchSize = batchSize
		o.BatchWait = batchWait

	}
}

func WithAuth(username, password string) func(*Options) {
	return func(o *Options) {
		o.Username = username
		o.Password = password
	}
}

func WithOthers(lokiTimeout int, honorOriginTime bool) func(*Options) {
	return func(o *Options) {
		o.LokiTimeout = lokiTimeout
		o.HonorOriginTime = honorOriginTime
	}
}

func NewLoki(URL string, customHeader map[string]string, opts ...func(*Options)) (*Loki, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	return NewLokiCustomHostname(URL, hostname, customHeader, opts...)
}

func NewLokiCustomHostname(URL, hostname string, customHeader map[string]string, opts ...func(*Options)) (*Loki, error) {

	options := &Options{
		BatchSize:       1000,
		BatchWait:       10,
		Username:        "",
		Password:        "",
		LokiTimeout:     10,
		HonorOriginTime: false,
	}

	for _, opt := range opts {
		opt(options)
	}

	l := &Loki{
		lokiURL:       URL,
		batchSize:     options.BatchSize,
		batchWait:     time.Duration(options.BatchWait) * time.Second,
		payloadCh:     make(chan payload, options.BatchSize),
		prependLabels: make(map[model.LabelName]model.LabelValue),
		hostname:      hostname,
		username:      options.Username,
		password:      options.Password,
		customHeader:  customHeader,
		lokiClient: &http.Client{
			Timeout: time.Duration(options.LokiTimeout) * time.Second,
		},
		honorOriginTime: options.HonorOriginTime,
	}

	u, err := url.Parse(l.lokiURL)
	if err != nil {
		return nil, err
	}
	if !strings.Contains(u.Path, postPath) {
		u.Path = postPath
		q := u.Query()
		u.RawQuery = q.Encode()
		l.lokiURL = u.String()
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
		maxWait     = time.NewTimer(l.batchWait)
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
			if !l.honorOriginTime && lastPktTime.After(curPktTime) {
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

			if batchSize+len(l.entry.Line) > l.batchSize {
				if err := l.sendBatch(batch); err != nil {
					fmt.Fprintf(os.Stderr, "%v ERROR: send size batch: %v\n", lastPktTime, err)
				}
				batchSize = 0
				batch = map[model.Fingerprint]*StreamAdapter{}
				maxWait.Reset(l.batchWait)
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
			maxWait.Reset(l.batchWait)
		}
	}
}

func (l *Loki) sendBatch(batch map[model.Fingerprint]*StreamAdapter) error {
	buf, err := encodeBatch(batch)
	if err != nil {
		return fmt.Errorf("encode batch error: %w", err)
	}

	// 重试相关配置
	maxRetries := 3
	backoffBase := time.Second

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		// 创建新的context，每次重试都有完整的超时时间
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		statusCode, err := l.send(ctx, buf)
		cancel() // 立即释放context资源

		if err == nil && statusCode >= 200 && statusCode < 300 {
			return nil // 发送成功
		}

		lastErr = err
		if err != nil {
			// 记录重试信息
			fmt.Fprintf(os.Stderr, "%v ERROR: send batch attempt %d failed: %v\n",
				time.Now(), attempt+1, err)
		} else {
			fmt.Fprintf(os.Stderr, "%v ERROR: send batch attempt %d failed with status code: %d\n",
				time.Now(), attempt+1, statusCode)
		}

		// 最后一次重试就不需要等待了
		if attempt < maxRetries-1 {
			// 使用指数退避策略，每次重试等待时间翻倍
			backoffTime := backoffBase * time.Duration(1<<uint(attempt))
			time.Sleep(backoffTime)
		}
	}

	// 所有重试都失败了
	return fmt.Errorf("failed to send batch after %d attempts, last error: %v",
		maxRetries, lastErr)
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
	req, err := http.NewRequest("POST", l.lokiURL, bytes.NewReader(buf))
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
	resp, err := l.lokiClient.Do(req)
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
