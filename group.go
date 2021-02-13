package chunk

import (
	"bytes"
	"io"
	"sort"
)

var _ sort.Interface = Group{}

// Group is a group of related chunks, these chunks are ordered by chunk's
// order (o) property. Missing chunks result in an error.
type Group []*Chunk

func (g Group) Len() int           { return len(g) }
func (g Group) Swap(i, j int)      { g[i], g[j] = g[j], g[i] }
func (g Group) Less(i, j int) bool { return g[i].o < g[j].o }

// MissingChunks sorts the chunks and returns true if any chunks are missing
func (g Group) MissingChunks() bool {
	for i := 0; i < len(g); i++ {
		if int64(i) != g[i].o {
			return true
		}
	}
	return false
}

func (g *Group) Add(chunk *Chunk) {
	*g = append(*g, chunk)
}

// Bytes reassembles the chunk group into a single byte slice
func (g Group) Bytes() []byte {
	buf := bytes.NewBuffer(nil)
	for _, chunk := range g {
		buf.Write(chunk.d)
	}
	return buf.Bytes()
}

// Reader returns an io.Reader that knows how to read the data from all the chunks in order
func (g Group) Reader() io.Reader {
	sort.Sort(g)
	buf := bytes.NewBuffer(nil)
	for _, chunk := range g {
		buf.Write(chunk.d)
	}
	return buf
}
