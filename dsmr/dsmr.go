package dsmr

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/alexbakker/go-dsmr/internal/crc16"
)

const (
	TimeFormat = "060102150405"
)

type Frame struct {
	Header      string
	Version     string
	EquipmentID string
	Time        time.Time
	Raw         string
	Objects     map[string]Object
}

type Reader struct {
	r *bufio.Reader
}

type Object struct {
	ID    string
	Value Value
	Time  time.Time
}

type Value struct {
	Data string
	Unit string
}

func NewReader(r io.Reader) *Reader {
	return &Reader{
		r: bufio.NewReader(r),
	}
}

func (r *Reader) Next() (*Frame, error) {
	for {
		b, err := r.r.Peek(1)
		if err != nil {
			return nil, fmt.Errorf("read dsmr header: %w", err)
		}

		if string(b) == "/" {
			break
		}

		// If we started reading somewhere in the middle of the stream, we need
		// to wait until we see a header
		if _, err = r.r.ReadByte(); err != nil {
			return nil, fmt.Errorf("read dsmr header: %w", err)
		}
	}

	rawFrame, err := r.r.ReadBytes('!')
	if err != nil {
		return nil, fmt.Errorf("read dsmr frame: %w", err)
	}

	rawCRC, err := r.r.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("read dsmr crc: %w", err)
	}

	crcBytes, err := hex.DecodeString(strings.TrimSpace(rawCRC))
	if err != nil {
		return nil, fmt.Errorf("parse dsmr crc: %w", err)
	}
	if len(crcBytes) != 2 {
		return nil, fmt.Errorf("parse dsmr crc: bad length: %d", len(crcBytes))
	}

	pcrc := binary.BigEndian.Uint16(crcBytes)
	crc := crc16.Checksum(rawFrame)

	if pcrc != crc {
		return nil, fmt.Errorf("dsmr crc mismatch: 0x%04X != 0x%04X", pcrc, crc)
	}

	return ParseFrame(string(rawFrame))
}

func ParseFrame(raw string) (*Frame, error) {
	f := Frame{
		Raw:     raw,
		Objects: make(map[string]Object),
	}

	for _, s := range strings.Split(raw, "\r\n") {
		if s == "" || s[0] == '!' {
			continue
		}

		if s[0] == '/' {
			f.Header = s[1:]
			continue
		}

		obj, err := ParseObject(s)
		if err != nil {
			return nil, err
		}
		f.Objects[obj.ID] = obj

		switch obj.ID {
		case "1-3:0.2.8":
			f.Version = obj.Value.Data
		case "0-0:1.0.0":
			if len(obj.Value.Data) != 0 {
				t, err := parseTimestamp(obj.Value.Data)
				if err != nil {
					return nil, fmt.Errorf("bad dsmr frame timestamp: %w", err)
				}
				f.Time = t
			}
		case "0-0:96.1.1":
			f.EquipmentID = obj.Value.Data
		}
	}

	return &f, nil
}

func ParseObject(raw string) (Object, error) {
	i := strings.Index(raw, "(")
	if i == -1 {
		return Object{}, fmt.Errorf("no values in dsmr object: %q", raw)
	}

	id := raw[:i]

	var values []Value
	for _, part := range strings.Split(raw[i:], "(") {
		if part == "" {
			continue
		}
		if !strings.HasSuffix(part, ")") {
			return Object{}, fmt.Errorf("bad value format in dsmr object: %q", raw)
		}

		value := strings.Split(part[:len(part)-1], "*")
		switch len(value) {
		case 1:
			values = append(values, Value{Data: value[0]})
		case 2:
			values = append(values, Value{Data: value[0], Unit: value[1]})
		default:
			return Object{}, fmt.Errorf("bad value unit format in dsmr object: %q", raw)
		}
	}

	var value Value
	var t time.Time
	switch len(values) {
	case 2:
		var err error
		t, err = parseTimestamp(values[0].Data)
		if err != nil {
			return Object{}, err
		}
		fallthrough
	case 1:
		value = values[len(values)-1]
	default:
		return Object{}, fmt.Errorf("unsupported number of values in dsmr object: %q", raw)
	}

	return Object{
		ID:    id,
		Value: value,
		Time:  t,
	}, nil
}

func parseTimestamp(s string) (time.Time, error) {
	if s == "0" {
		return time.Time{}, nil
	}

	loc, err := time.LoadLocation("Europe/Amsterdam")
	if err != nil {
		return time.Time{}, err
	}

	// Strip the S/W suffix and parse the timestamp
	if !strings.HasSuffix(s, "S") && !strings.HasSuffix(s, "W") {
		return time.Time{}, fmt.Errorf("bad dsmr frame timestamp: %w", err)
	}
	t, err := time.ParseInLocation(TimeFormat, s[:len(s)-1], loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("bad dsmr frame timestamp: %w", err)
	}

	return t, nil
}
