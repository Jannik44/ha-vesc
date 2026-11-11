package main

// Minimale VESC-Protokoll-Implementierung (TCP über VESC Express).
//
// Paketformat:
//   kurz:  0x02 | len(1) | payload | crc16(2) | 0x03
//   lang:  0x03 | len(2) | payload | crc16(2) | 0x03
// CRC16 = CRC-16/XMODEM (Poly 0x1021, Init 0) über den Payload.

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"
)

const (
	commGetValues  = 4
	commForwardCAN = 34
	commPingCAN    = 62
)

type Values struct {
	Voltage   float64 `json:"voltage"`
	SOC       float64 `json:"soc"`
	TempESC   float64 `json:"temp_esc"`
	TempMotor float64 `json:"temp_motor"`
	CanID     int     `json:"can_id"`
}

func crc16(data []byte) uint16 {
	var crc uint16
	for _, b := range data {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = crc<<1 ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

func buildPacket(payload []byte) []byte {
	var pkt []byte
	if len(payload) <= 255 {
		pkt = []byte{0x02, byte(len(payload))}
	} else {
		pkt = []byte{0x03, byte(len(payload) >> 8), byte(len(payload))}
	}
	pkt = append(pkt, payload...)
	pkt = binary.BigEndian.AppendUint16(pkt, crc16(payload))
	return append(pkt, 0x03)
}

// packetReader sammelt TCP-Daten und liefert vollständige, CRC-geprüfte Payloads.
type packetReader struct{ buf []byte }

func (r *packetReader) feed(data []byte) [][]byte {
	r.buf = append(r.buf, data...)
	var out [][]byte
	for len(r.buf) > 0 {
		var hlen int
		switch r.buf[0] {
		case 0x02:
			hlen = 1
		case 0x03:
			hlen = 2
		default:
			r.buf = r.buf[1:] // bis zum nächsten Startbyte verwerfen
			continue
		}
		if len(r.buf) < 1+hlen {
			break
		}
		plen := int(r.buf[1])
		if hlen == 2 {
			plen = int(binary.BigEndian.Uint16(r.buf[1:3]))
		}
		if plen == 0 || plen > 4096 {
			r.buf = r.buf[1:]
			continue
		}
		total := 1 + hlen + plen + 3
		if len(r.buf) < total {
			break
		}
		payload := r.buf[1+hlen : 1+hlen+plen]
		crc := binary.BigEndian.Uint16(r.buf[1+hlen+plen:])
		if r.buf[total-1] == 0x03 && crc == crc16(payload) {
			out = append(out, append([]byte(nil), payload...))
			r.buf = r.buf[total:]
		} else {
			r.buf = r.buf[1:] // resync
		}
	}
	return out
}

type vescConn struct {
	conn   net.Conn
	reader packetReader
}

// request sendet payload und wartet auf eine Antwort mit Kommando-Byte expect.
// Gibt nil zurück, wenn innerhalb des Timeouts nichts Passendes kam.
func (c *vescConn) request(payload []byte, expect byte, timeout time.Duration) ([]byte, error) {
	if _, err := c.conn.Write(buildPacket(payload)); err != nil {
		return nil, err
	}
	deadline := time.Now().Add(timeout)
	buf := make([]byte, 4096)
	for time.Now().Before(deadline) {
		c.conn.SetReadDeadline(deadline)
		n, err := c.conn.Read(buf)
		if err != nil {
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				return nil, nil
			}
			return nil, err
		}
		for _, pkt := range c.reader.feed(buf[:n]) {
			if pkt[0] == expect {
				return pkt[1:], nil
			}
		}
	}
	return nil, nil
}

func parseGetValues(p []byte) (Values, error) {
	if len(p) < 28 {
		return Values{}, fmt.Errorf("GET_VALUES Antwort zu kurz (%d Bytes)", len(p))
	}
	i16 := func(off int) float64 { return float64(int16(binary.BigEndian.Uint16(p[off:]))) }
	return Values{
		TempESC:   i16(0) / 10,
		TempMotor: i16(2) / 10,
		Voltage:   i16(26) / 10,
	}, nil
}

// ReadVESC liest die Werte über den VESC Express.
// canID < 0: CAN-ID per COMM_PING_CAN automatisch ermitteln.
func ReadVESC(host string, port, canID int) (Values, error) {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), 10*time.Second)
	if err != nil {
		return Values{}, err
	}
	defer conn.Close()
	c := &vescConn{conn: conn}

	ids := []int{canID}
	if canID < 0 {
		resp, err := c.request([]byte{commPingCAN}, commPingCAN, 12*time.Second)
		if err != nil {
			return Values{}, err
		}
		if resp == nil {
			return Values{}, errors.New("COMM_PING_CAN Timeout")
		}
		if len(resp) == 0 {
			return Values{}, errors.New("kein VESC am CAN-Bus gefunden")
		}
		ids = ids[:0]
		for _, id := range resp {
			ids = append(ids, int(id))
		}
	}

	for attempt := 0; attempt < 6; attempt++ {
		id := ids[attempt%len(ids)]
		resp, err := c.request([]byte{commForwardCAN, byte(id), commGetValues}, commGetValues, 2*time.Second)
		if err != nil {
			return Values{}, err
		}
		if resp != nil {
			v, err := parseGetValues(resp)
			v.CanID = id
			return v, err
		}
	}
	return Values{}, fmt.Errorf("keine GET_VALUES Antwort (CAN IDs %v)", ids)
}
