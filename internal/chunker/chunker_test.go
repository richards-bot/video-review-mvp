package chunker

import (
	"bytes"
	"io"
	"testing"
)

func TestChunker_EmptyFile(t *testing.T) {
	c := New(bytes.NewReader(nil), 1024)
	_, err := c.Next()
	if err != io.EOF {
		t.Errorf("empty reader: expected io.EOF, got %v", err)
	}
}

func TestChunker_SmallerThanChunkSize(t *testing.T) {
	data := []byte("hello world")
	c := New(bytes.NewReader(data), 1024)

	chunk, err := c.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if chunk.Index != 0 {
		t.Errorf("chunk.Index = %d, want 0", chunk.Index)
	}
	if !bytes.Equal(chunk.Data, data) {
		t.Errorf("chunk.Data = %q, want %q", chunk.Data, data)
	}

	_, err = c.Next()
	if err != io.EOF {
		t.Errorf("after last chunk: expected io.EOF, got %v", err)
	}
}

func TestChunker_ExactMultiple(t *testing.T) {
	chunkSize := 4
	data := []byte("abcdefgh") // 8 bytes = 2 * chunkSize
	chunks, err := ReadAll(bytes.NewReader(data), chunkSize)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(chunks) != 2 {
		t.Fatalf("got %d chunks, want 2", len(chunks))
	}
	if !bytes.Equal(chunks[0].Data, []byte("abcd")) {
		t.Errorf("chunk 0 = %q, want %q", chunks[0].Data, "abcd")
	}
	if !bytes.Equal(chunks[1].Data, []byte("efgh")) {
		t.Errorf("chunk 1 = %q, want %q", chunks[1].Data, "efgh")
	}
	if chunks[0].Index != 0 || chunks[1].Index != 1 {
		t.Errorf("wrong indices: %d %d", chunks[0].Index, chunks[1].Index)
	}
}

func TestChunker_ExactMultiplePlusOne(t *testing.T) {
	chunkSize := 4
	data := []byte("abcdefghi") // 9 bytes = 2 * chunkSize + 1
	chunks, err := ReadAll(bytes.NewReader(data), chunkSize)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(chunks) != 3 {
		t.Fatalf("got %d chunks, want 3", len(chunks))
	}
	if !bytes.Equal(chunks[2].Data, []byte("i")) {
		t.Errorf("final chunk = %q, want %q", chunks[2].Data, "i")
	}
}

func TestChunker_ManyChunks(t *testing.T) {
	chunkSize := 10
	data := bytes.Repeat([]byte("x"), 100)
	chunks, err := ReadAll(bytes.NewReader(data), chunkSize)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(chunks) != 10 {
		t.Fatalf("got %d chunks, want 10", len(chunks))
	}
	for i, ch := range chunks {
		if uint64(i) != ch.Index {
			t.Errorf("chunk %d has index %d", i, ch.Index)
		}
		if len(ch.Data) != chunkSize {
			t.Errorf("chunk %d size = %d, want %d", i, len(ch.Data), chunkSize)
		}
	}
}

func TestChunker_SingleByte(t *testing.T) {
	chunks, err := ReadAll(bytes.NewReader([]byte{0x42}), 1024)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("got %d chunks, want 1", len(chunks))
	}
	if chunks[0].Data[0] != 0x42 {
		t.Error("wrong data")
	}
}
