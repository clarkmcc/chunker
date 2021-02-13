package chunk

import (
	"bufio"
	"io"
)

// Chunker knows how to chunk a file and returns a channel of chunks and errors
type Chunker interface {
	Chunk(r io.Reader) (<-chan *Chunk, <-chan error)
}

type ChunkerOpts struct {
	// The size in bytes of each chunk
	ChunkSize int
	// The number of chunks that can be buffered onto the chunk chan
	BufferSize int
}

func (o *ChunkerOpts) WithDefaults() *ChunkerOpts {
	if o.ChunkSize <= 0 {
		// 4kb chunks by default
		o.ChunkSize = 4 * 1024
	}
	if o.BufferSize <= 0 {
		// buffer up to 1024 chunks at a time
		o.BufferSize = 1024
	}
	return o
}

var _ Chunker = &ChunkerImpl{}

type ChunkerImpl struct {
	size   int
	buffer int
}

func (c *ChunkerImpl) Chunk(r io.Reader) (<-chan *Chunk, <-chan error) {
	ch := make(chan *Chunk, c.buffer)
	ech := make(chan error)
	go chunkTo(r, ch, ech, c.size)
	return ch, ech
}

// chunkTo chunks the data read from s to the provided chan to, if any errors
// are returned, they are returned to the provided errs chan.
func chunkTo(s io.Reader, to chan *Chunk, errs chan error, size int) {
	defer close(to)
	err := chunkAndDo(s, size, func(chunk *Chunk) {
		to <- chunk
	})
	if err != nil {
		errs <- err
	}
}

// chunkAndDo chunks the data read from s into chunks of the provided size
// calling do for each chunk created
func chunkAndDo(s io.Reader, size int, do func(*Chunk)) error {
	r := bufio.NewReader(s)
	i := 0
	for {
		buf := make([]byte, 0, size)
		n, err := r.Read(buf[:cap(buf)])
		buf = buf[:n]
		if n == 0 {
			if err == nil {
				continue
			}
			if err == io.EOF {
				break
			}
			return err
		}
		// Do something with buf
		do(makeChunk(i, buf))
		i++
	}
	return nil
}

func NewChunker(opts *ChunkerOpts) *ChunkerImpl {
	if opts == nil {
		opts = (&ChunkerOpts{}).WithDefaults()
	}
	opts = opts.WithDefaults()
	return &ChunkerImpl{
		size:   opts.ChunkSize,
		buffer: opts.BufferSize,
	}
}
