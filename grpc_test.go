package chunk

import (
	"bytes"
	"fmt"
	"github.com/clarkmcc/chunker/protos"
	"github.com/stretchr/testify/require"
	"io"
	"sync"
	"testing"
)

var _ WrappedServer = &fakeWrappedServer{}

type fakeWrappedServer struct {
	numChunks   int
	i           int
	shouldError bool
	closed      bool
	lock        sync.Mutex
}

func (m *fakeWrappedServer) Receive() (*protos.Chunk, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.closed {
		return nil, io.ErrClosedPipe
	}
	if m.shouldError {
		return nil, fmt.Errorf("intentional error")
	}
	if m.i >= m.numChunks {
		return nil, io.EOF
	}
	defer func() {
		m.i++
	}()
	return &protos.Chunk{O: int64(m.i)}, nil
}

func (m *fakeWrappedServer) Close() error {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.closed = true
	return nil
}

func newFakeWrappedServer(n int, shouldError, closed bool) *fakeWrappedServer {
	return &fakeWrappedServer{
		numChunks:   n,
		shouldError: shouldError,
		closed:      closed,
	}
}

var _ WrappedClient = &fakeWrappedClient{}

type fakeWrappedClient struct {
	chunks      []*protos.Chunk
	lock        sync.Mutex
	shouldError bool
	closed      bool
}

func (f *fakeWrappedClient) Send(chunk *protos.Chunk) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	if f.shouldError {
		return fmt.Errorf("intentional error")
	}
	if f.closed {
		return io.ErrClosedPipe
	}
	f.chunks = append(f.chunks, chunk)
	return nil
}

func (f *fakeWrappedClient) Close() error {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.closed = true
	return nil
}

func newFakeWrappedClient(shouldError, closed bool) *fakeWrappedClient {
	return &fakeWrappedClient{
		chunks:      []*protos.Chunk{},
		shouldError: shouldError,
		closed:      closed,
	}
}

func makeBufferWithData(size int) *bytes.Buffer {
	buf := bytes.NewBuffer(nil)
	for i := 0; i < size; i++ {
		buf.Write([]byte{0})
	}
	return buf
}

func TestCollectorImpl_Collect(t *testing.T) {
	t.Run("Normal", func(t *testing.T) {
		group, err := NewCollector().Collect(newFakeWrappedServer(10, false, false))
		require.NoError(t, err)
		require.Equal(t, 10, group.Len())
	})
	t.Run("Error", func(t *testing.T) {
		group, err := NewCollector().Collect(newFakeWrappedServer(10, true, false))
		require.Error(t, err)
		require.Nil(t, group)
	})
	t.Run("Closed", func(t *testing.T) {
		group, err := NewCollector().Collect(newFakeWrappedServer(10, false, true))
		require.Error(t, err)
		require.Nil(t, group)
	})
}

func TestUploadFrom(t *testing.T) {
	t.Run("Normal", func(t *testing.T) {
		client := newFakeWrappedClient(false, false)
		err := UploadFrom(makeBufferWithData(100*1024), client)
		require.NoError(t, err)
		// Data size is 1024*100 chunked in 4096 byte chunks
		require.Equal(t, 25, len(client.chunks))
	})
	t.Run("Error", func(t *testing.T) {
		client := newFakeWrappedClient(true, false)
		err := UploadFrom(makeBufferWithData(100*1024), client)
		require.Error(t, err)
		// Data size is 1024*100 chunked in 4096 byte chunks
		require.Equal(t, 0, len(client.chunks))
	})
	t.Run("Closed", func(t *testing.T) {
		client := newFakeWrappedClient(false, true)
		err := UploadFrom(makeBufferWithData(100*1024), client)
		require.Error(t, err)
		// Data size is 1024*100 chunked in 4096 byte chunks
		require.Equal(t, 0, len(client.chunks))
	})
}
