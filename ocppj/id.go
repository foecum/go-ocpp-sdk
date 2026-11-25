package ocppj

import (
	"crypto/rand"
	"encoding/hex"
)

// IDGenerator creates unique message IDs of at most MaxIDLength characters.
// Implementations must be safe for concurrent use.
type IDGenerator interface {
	NewID() string
}

// IDGeneratorFunc adapts a function to IDGenerator.
type IDGeneratorFunc func() string

// NewID calls f.
func (f IDGeneratorFunc) NewID() string { return f() }

type uuidGenerator struct{}

// NewUUIDGenerator returns the default generator, producing random (version 4)
// UUIDs such as "3f2b8c1e-5a4d-4e0b-9c7a-1d2e3f4a5b6c" (36 characters).
func NewUUIDGenerator() IDGenerator { return uuidGenerator{} }

func (uuidGenerator) NewID() string {
	var b [16]byte
	// As of Go 1.24 crypto/rand.Read never returns an error.
	_, _ = rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40 // version 4
	b[8] = b[8]&0x3f | 0x80 // RFC 4122 variant
	var out [36]byte
	hex.Encode(out[0:8], b[0:4])
	out[8] = '-'
	hex.Encode(out[9:13], b[4:6])
	out[13] = '-'
	hex.Encode(out[14:18], b[6:8])
	out[18] = '-'
	hex.Encode(out[19:23], b[8:10])
	out[23] = '-'
	hex.Encode(out[24:], b[10:])
	return string(out[:])
}
