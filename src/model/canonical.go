package model

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

const (
	CanonicalEncodingVersion = "agentrix-canonical/1"
)

// CanonicalEncoder implements the binary canonical encoder for agentrix-canonical/1.
// Every field is length-delimited or fixed-width, integers are big-endian,
// and floats reject non-finite values and normalize negative zero.
type CanonicalEncoder struct {
	buf []byte
}

func NewCanonicalEncoder(domain string) *CanonicalEncoder {
	e := &CanonicalEncoder{
		buf: make([]byte, 0, 256),
	}
	e.String(CanonicalEncodingVersion)
	e.String(domain)
	return e
}

func (e *CanonicalEncoder) Bytes() []byte {
	return e.buf
}

func (e *CanonicalEncoder) U8(v uint8) {
	e.buf = append(e.buf, v)
}

func (e *CanonicalEncoder) Bool(v bool) {
	if v {
		e.U8(1)
	} else {
		e.U8(0)
	}
}

func (e *CanonicalEncoder) U32(v uint32) {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], v)
	e.buf = append(e.buf, b[:]...)
}

func (e *CanonicalEncoder) U64(v uint64) {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], v)
	e.buf = append(e.buf, b[:]...)
}

func (e *CanonicalEncoder) I64(v int64) {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(v))
	e.buf = append(e.buf, b[:]...)
}

func (e *CanonicalEncoder) RawBytes(v []byte) {
	e.U64(uint64(len(v)))
	e.buf = append(e.buf, v...)
}

func (e *CanonicalEncoder) String(v string) {
	e.RawBytes([]byte(v))
}

func (e *CanonicalEncoder) F32(v float32) error {
	if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
		return fmt.Errorf("canonical f32 must be finite, got %v", v)
	}
	if v == 0.0 {
		v = 0.0
	}
	e.U32(math.Float32bits(v))
	return nil
}

func (e *CanonicalEncoder) F64(v float64) error {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return fmt.Errorf("canonical f64 must be finite, got %v", v)
	}
	if v == 0.0 {
		v = 0.0
	}
	e.U64(math.Float64bits(v))
	return nil
}

func (e *CanonicalEncoder) JSON(v interface{}) error {
	switch val := v.(type) {
	case nil:
		e.U8(0)
	case bool:
		e.U8(1)
		e.Bool(val)
	case json.Number:
		e.U8(2)
		s := val.String()
		if strings.ContainsAny(s, ".eE") {
			f, err := val.Float64()
			if err != nil {
				return fmt.Errorf("invalid json float number %q: %w", s, err)
			}
			e.U8(2)
			if err := e.F64(f); err != nil {
				return err
			}
		} else if strings.HasPrefix(s, "-") {
			i, err := val.Int64()
			if err != nil {
				return fmt.Errorf("invalid json signed int %q: %w", s, err)
			}
			e.U8(1)
			e.I64(i)
		} else {
			u, err := strconv.ParseUint(s, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid json unsigned int %q: %w", s, err)
			}
			e.U8(0)
			e.U64(u)
		}
	case float64:
		e.U8(2)
		e.U8(2)
		if err := e.F64(val); err != nil {
			return err
		}
	case float32:
		e.U8(2)
		e.U8(2)
		if err := e.F64(float64(val)); err != nil {
			return err
		}
	case int:
		e.U8(2)
		if val >= 0 {
			e.U8(0)
			e.U64(uint64(val))
		} else {
			e.U8(1)
			e.I64(int64(val))
		}
	case int64:
		e.U8(2)
		if val >= 0 {
			e.U8(0)
			e.U64(uint64(val))
		} else {
			e.U8(1)
			e.I64(val)
		}
	case uint64:
		e.U8(2)
		e.U8(0)
		e.U64(val)
	case uint32:
		e.U8(2)
		e.U8(0)
		e.U64(uint64(val))
	case string:
		e.U8(3)
		e.String(val)
	case []interface{}:
		e.U8(4)
		e.U64(uint64(len(val)))
		for _, item := range val {
			if err := e.JSON(item); err != nil {
				return err
			}
		}
	case map[string]interface{}:
		e.U8(5)
		e.U64(uint64(len(val)))
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			e.String(k)
			if err := e.JSON(val[k]); err != nil {
				return err
			}
		}
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("unsupported json value: %T", v)
		}
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		var normalized interface{}
		if err := dec.Decode(&normalized); err != nil {
			return err
		}
		return e.JSON(normalized)
	}
	return nil
}

// SHA256Hex computes the domain-separated SHA256 hexadecimal digest matching Rust's sha256_hex.
func SHA256Hex(domain string, parts ...[]byte) string {
	h := sha256.New()
	var domainLenBuf [8]byte
	binary.BigEndian.PutUint64(domainLenBuf[:], uint64(len(domain)))
	h.Write(domainLenBuf[:])
	h.Write([]byte(domain))
	for _, part := range parts {
		var partLenBuf [8]byte
		binary.BigEndian.PutUint64(partLenBuf[:], uint64(len(part)))
		h.Write(partLenBuf[:])
		h.Write(part)
	}
	sum := h.Sum(nil)
	return hex.EncodeToString(sum)
}

// CanonicalJSONDigest computes the canonical JSON digest using CanonicalEncoder and SHA256Hex.
func CanonicalJSONDigest(domain string, value interface{}) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal json for canonical digest: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var parsed interface{}
	if err := dec.Decode(&parsed); err != nil {
		return "", fmt.Errorf("decode json for canonical digest: %w", err)
	}
	encoder := NewCanonicalEncoder(domain)
	if err := encoder.JSON(parsed); err != nil {
		return "", err
	}
	return SHA256Hex(domain, encoder.Bytes()), nil
}
