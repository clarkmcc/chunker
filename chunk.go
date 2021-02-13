package chunk

type Chunk struct {
	// used to preserve chunk order
	o int64
	// the data contained in the chunk
	d []byte
}

// makeChunk creates a new chunk based on the provided index and byte slice data
func makeChunk(i int, d []byte) *Chunk {
	return &Chunk{
		o: int64(i),
		d: d,
	}
}
