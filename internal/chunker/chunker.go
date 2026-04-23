// Package chunker provides streaming chunked reading from an io.Reader.
package chunker

import (
	"io"
)

// Chunk represents a single chunk of data with its index.
type Chunk struct {
	Index uint64
	Data  []byte
}

// Chunker reads from a source in fixed-size chunks.
type Chunker struct {
	r         io.Reader
	chunkSize int
	index     uint64
}

// New creates a Chunker reading from r in chunkSize-byte pieces.
func New(r io.Reader, chunkSize int) *Chunker {
	if chunkSize <= 0 {
		chunkSize = 1024 * 1024
	}
	return &Chunker{r: r, chunkSize: chunkSize}
}

// Next returns the next chunk. Returns (Chunk{}, io.EOF) when the reader
// is exhausted. The final chunk may be smaller than chunkSize.
// Returns an error if reading fails (not including io.EOF at chunk boundary).
func (c *Chunker) Next() (Chunk, error) {
	buf := make([]byte, c.chunkSize)
	n, err := io.ReadFull(c.r, buf)
	if n > 0 {
		chunk := Chunk{
			Index: c.index,
			Data:  buf[:n],
		}
		c.index++
		if err == io.ErrUnexpectedEOF || err == nil {
			return chunk, nil
		}
		return chunk, err
	}
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return Chunk{}, io.EOF
	}
	return Chunk{}, err
}

// ReadAll reads all chunks from the reader and returns them.
func ReadAll(r io.Reader, chunkSize int) ([]Chunk, error) {
	c := New(r, chunkSize)
	var chunks []Chunk
	for {
		chunk, err := c.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, chunk)
	}
	return chunks, nil
}
