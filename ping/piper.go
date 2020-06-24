package ping

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"net"
	"sort"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

type RoundTrip struct {
	Sequence    int           `json:"sequence"`
	PayloadSize int           `json:"payloadSize"`
	Target      net.IP        `json:"target"`
	TTL         int           `json:"TTL"`
	Error       error         `json:"error"`
	EmittedAt   time.Time     `json:"-"`
	RTT         time.Duration `json:"RTT"`
}

func (rt RoundTrip) String() string {
	return fmt.Sprintf("%d bytes from %s: seq=%d ttl=%d time=%.3f ms",
		rt.PayloadSize, rt.Target, rt.Sequence, rt.TTL, rt.RTT.Seconds()*1000)
}

type piper struct {
	ipAddr           net.IPAddr
	conn             *icmp.PacketConn
	isV4             bool
	stopCh           chan struct{}
	checkTimes       uint
	roundTripTimeout time.Duration
	size             int
	sequence         int
	spanID           int
	RTs              map[int]RoundTrip
}

const (
	maxPiperPacketSize = 512
	piperSpanSize      = 5
	piperSeqSize       = 8

	ProtocolICMP     = 1  // Internet Control Message
	ProtocolIPv6ICMP = 58 // ICMP for IPv6
)

func newPiper(ipAddr net.IPAddr, conn *icmp.PacketConn, isV4 bool,
	checkTimes uint, roundTripTimeout time.Duration, size int) *piper {
	return &piper{
		ipAddr:           ipAddr,
		conn:             conn,
		size:             size,
		isV4:             isV4,
		stopCh:           make(chan struct{}),
		checkTimes:       checkTimes,
		roundTripTimeout: roundTripTimeout,
		spanID:           rand.Intn(math.MaxInt16),
		RTs:              make(map[int]RoundTrip, checkTimes),
		sequence:         0,
	}
}

func (p *piper) Run() []RoundTrip {
	for p.checkTimes > 0 {
		err := p.sendMessage()
		if err == nil {
			// Try receive
			p.receivePacket()
		}
		p.checkTimes--
		p.sequence++
	}
	var RTs []RoundTrip
	for _, RT := range p.RTs {
		RTs = append(RTs, RT)
	}
	sort.Slice(RTs, func(i, j int) bool {
		return RTs[i].Sequence < RTs[j].Sequence
	})
	return RTs
}

func (p *piper) sendMessage() error {
	// Construct echo data
	payload := append(intToBytes(int64(p.sequence)), intToBytes(int64(p.spanID))...)
	// Get remain payload space and fill them
	if remainLength := p.size - piperSeqSize - piperSpanSize; remainLength > 0 {
		payload = append(payload, bytes.Repeat([]byte{0}, remainLength)...)
	}
	messageType := icmp.Type(ipv4.ICMPTypeEcho)
	if !p.isV4 {
		messageType = ipv6.ICMPTypeEchoRequest
	}
	message := &icmp.Message{
		Type: messageType,
		Code: 0,
		Body: &icmp.Echo{
			ID:   p.spanID,
			Seq:  p.sequence,
			Data: payload,
		},
	}

	messagePayload, err := message.Marshal(nil)
	if err == nil {
		_, err = p.conn.WriteTo(messagePayload, &p.ipAddr)
	}
	p.RTs[p.sequence] = RoundTrip{
		Sequence:  p.sequence,
		Target:    p.ipAddr.IP,
		Error:     err,
		EmittedAt: time.Now(),
	}
	return err
}

func (p *piper) receivePacket() {
	var (
		size int
		err  error
	)
	// read from connection
	p.conn.SetReadDeadline(time.Now().Add(p.roundTripTimeout))
	payload := make([]byte, maxPiperPacketSize)
	var ttl int
	if p.isV4 {
		var cm *ipv4.ControlMessage
		size, cm, _, err = p.conn.IPv4PacketConn().ReadFrom(payload)
		if cm != nil {
			ttl = cm.TTL
		}
	} else {
		var cm *ipv6.ControlMessage
		size, cm, _, err = p.conn.IPv6PacketConn().ReadFrom(payload)
		if cm != nil {
			ttl = cm.HopLimit
		}
	}
	receivedAt := time.Now()
	RT, ok := p.RTs[p.sequence]
	if !ok {
		return
	}
	dropped := false
	defer func() {
		if !dropped {
			p.RTs[p.sequence] = RT
		}
	}()
	if err != nil {
		RT.Error = err
		return
	}

	icmpProto := ProtocolICMP
	if !p.isV4 {
		icmpProto = ProtocolIPv6ICMP
	}
	message, err := icmp.ParseMessage(icmpProto, payload)
	if err != nil {
		RT.Error = err
		return
	}
	switch data := message.Body.(type) {
	case *icmp.Echo:
		if data.ID != p.spanID || data.Seq != p.sequence {
			dropped = true
			return
		}
		if len(data.Data) < p.size {
			RT.Error = fmt.Errorf("invalid response size: %d(expecte %d)", len(data.Data), p.size)
		}
		// Ignore currently data validate
		RT.TTL = ttl
		RT.PayloadSize = size
		RT.RTT = receivedAt.Sub(RT.EmittedAt)
	default:
		// Unexpected data
		dropped = true
		return
	}
}

func intToBytes(i int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(i))
	return b
}

func bytesToInt(b []byte) int64 {
	return int64(binary.BigEndian.Uint64(b))
}
