package chunk

import (
	"errors"
	"fmt"
	"github.com/clarkmcc/chunker/protos"
	"io"
	"sort"
)

// WrappedClient knows how to wrap an RPC client and provides an easy to
// use interface for the UploadFile function to interact with multiple rpc
// endpoint.
type WrappedClient interface {
	Send(chunk *protos.Chunk) error
	io.Closer
}

// WrappedServer knows how to wrap an RPC chunking server and provides an easy
// interface for a chunk collector to get all the chunks needed.
type WrappedServer interface {
	Receive() (*protos.Chunk, error)
	io.Closer
}

// ToRPC converts an internal chunk into an RPC chunk that can be serialized
// and deserialized by the grpc client and server.
func ToRPC(chunk *Chunk) *protos.Chunk {
	return &protos.Chunk{
		O: chunk.o,
		D: chunk.d,
	}
}

// ToRPC converts an RPC chunk into an internal chunk that can be serialized
// and deserialized by the grpc client and server.
func FromRPC(chunk *protos.Chunk) *Chunk {
	return &Chunk{
		o: chunk.O,
		d: chunk.D,
	}
}

// UploadFrom uploads the data from the provided reader to the provided chunking client
func UploadFrom(r io.Reader, client WrappedClient) error {
	chunker := NewChunker(&ChunkerOpts{})
	err := NewFunctionHandler(func(chunk *Chunk) error {
		return client.Send(ToRPC(chunk))
	}).HandleChunks(chunker.Chunk(r))
	if err != nil {
		return err
	}
	return client.Close()
}

// Collector knows how to collect chunks from a wrapped chunk stream server and return
// a chunk group. The chunk group is guaranteed to be complete and in order.
type Collector interface {
	Collect(srv WrappedServer) (*Group, error)
}

var _ Collector = &CollectorImpl{}

type CollectorImpl struct{}

// Collect collects chunks from the provided chunk stream server and groups the
// chunks together until an EOF is received. The chunk group is sorted, and checked
// for missing chunks before being returned.
func (c *CollectorImpl) Collect(srv WrappedServer) (*Group, error) {
	group := &Group{}
	for {
		chunk, err := srv.Receive()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		group.Add(FromRPC(chunk))
	}
	sort.Sort(group)
	if group.MissingChunks() {
		return nil, fmt.Errorf("detected missing chunks, please retry")
	}
	return group, srv.Close()
}

func NewCollector() *CollectorImpl {
	return &CollectorImpl{}
}
