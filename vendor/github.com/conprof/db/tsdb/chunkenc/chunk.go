// Copyright 2017 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package chunkenc

import (
	"math"
	"sync"

	"github.com/pkg/errors"
)

// Encoding is the identifier for a chunk encoding.
type Encoding uint8

func (e Encoding) String() string {
	switch e {
	case EncNone:
		return "none"
	case EncXOR:
		return "XOR"
	case EncBytes:
		return "Bytes"
	case EncTimestamps:
		return "Timestamps"
	case EncValues:
		return "Values"
	default:
		return "<unknown>"
	}
}

// The different available chunk encodings.
const (
	EncNone Encoding = iota
	EncBytes
	EncTimestamps
	EncValues
	EncXOR
)

// Chunk holds a sequence of sample pairs that can be iterated over and appended to.
type Chunk interface {
	// Bytes returns the underlying byte slice of the chunk.
	Bytes() ([]byte, error)

	// Encoding returns the encoding type of the chunk.
	Encoding() Encoding

	// Appender returns an appender to append samples to the chunk.
	Appender() (Appender, error)

	// The iterator passed as argument is for re-use.
	// Depending on implementation, the iterator can
	// be re-used or a new iterator can be allocated.
	Iterator(Iterator) Iterator

	// NumSamples returns the number of samples in the chunk.
	NumSamples() int

	// Compact is called whenever a chunk is expected to be complete (no more
	// samples appended) and the underlying implementation can eventually
	// optimize the chunk.
	// There's no strong guarantee that no samples will be appended once
	// Compact() is called. Implementing this function is optional.
	Compact()
}

// Appender adds sample pairs to a chunk.
type Appender interface {
	Append(int64, []byte)
}

// Iterator is a simple iterator that can only get the next value.
// Iterator iterates over the samples of a time series, in timestamp-increasing order.
type Iterator interface {
	// Next advances the iterator by one.
	Next() bool
	// Seek advances the iterator forward to the first sample with the timestamp equal or greater than t.
	// If current sample found by previous `Next` or `Seek` operation already has this property, Seek has no effect.
	// Seek returns true, if such sample exists, false otherwise.
	// Iterator is exhausted when the Seek returns false.
	Seek(t int64) bool
	// At returns the current timestamp/value pair.
	// Before the iterator has advanced At behavior is unspecified.
	At() (int64, []byte)
	// Err returns the current error. It should be used only after iterator is
	// exhausted, that is `Next` or `Seek` returns false.
	Err() error
}

// NewNopIterator returns a new chunk iterator that does not hold any data.
func NewNopIterator() Iterator {
	return nopIterator{}
}

type nopIterator struct{}

func (nopIterator) Seek(int64) bool     { return false }
func (nopIterator) At() (int64, []byte) { return math.MinInt64, []byte{} }
func (nopIterator) Next() bool          { return false }
func (nopIterator) Err() error          { return nil }

// Pool is used to create and reuse chunk references to avoid allocations.
type Pool interface {
	Put(Chunk) error
	Get(e Encoding, b []byte) (Chunk, error)
}

// pool is a memory pool of chunk objects.
type pool struct {
	xor sync.Pool
}

// NewPool returns a new pool.
func NewPool() Pool {
	return &pool{
		xor: sync.Pool{
			New: func() interface{} {
				return NewBytesChunk()
			},
		},
	}
}

func (p *pool) Get(e Encoding, b []byte) (Chunk, error) {
	switch e {
	case EncXOR:
		panic("nothing should be using XOR encoding")
	//	c := p.xor.Get().(*XORChunk)
	//	c.b.stream = b
	//	c.b.count = 0
	//	return c, nil

	// needs new case for our new encoding
	// c.immutable needs to be set = true
	// (need to double check also where else this Pool is used to see if this immutable thing is safe to do)
	case EncBytes:
		return LoadBytesChunk(b), nil
	}
	return nil, errors.Errorf("invalid chunk encoding %q", e)
}

func (p *pool) Put(c Chunk) error {
	switch c.Encoding() {
	case EncXOR:
		panic("nothing should be using XOR encoding")
	//	xc, ok := c.(*XORChunk)
	//	// This may happen often with wrapped chunks. Nothing we can really do about
	//	// it but returning an error would cause a lot of allocations again. Thus,
	//	// we just skip it.
	//	if !ok {
	//		return nil
	//	}
	//	xc.b.stream = nil
	//	xc.b.count = 0
	//	p.xor.Put(c)
	case EncBytes:
		xc, ok := c.(*BytesChunk)
		// This may happen often with wrapped chunks. Nothing we can really do about
		// it but returning an error would cause a lot of allocations again. Thus,
		// we just skip it.
		if !ok {
			return nil
		}
		xc.b = nil
		p.xor.Put(c)
	// needs new case for new encoding
	// needs to reset chunk.immutable = false
	default:
		return errors.Errorf("invalid chunk encoding %q", c.Encoding())
	}
	return nil
}

// FromData returns a chunk from a byte slice of chunk data.
// This is there so that users of the library can easily create chunks from
// bytes.
func FromData(e Encoding, d []byte) (Chunk, error) {
	switch e {
	case EncXOR:
		panic("nothing should be using XOR encoding")
	//	return &XORChunk{b: bstream{count: 0, stream: d}}, nil
	case EncBytes:
		return LoadBytesChunk(d), nil
	}
	return nil, errors.Errorf("invalid chunk encoding %q", e)
}
