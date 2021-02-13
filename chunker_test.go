package chunk

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
)

func makeFile(t *testing.T, size int) (file *os.File, cleanup func()) {
	buf := bytes.NewBuffer(nil)
	for i := 0; i < size; i++ {
		buf.Write([]byte{0})
	}
	file, err := os.Create("/tmp/test")
	require.NoError(t, err)
	file.Write(buf.Bytes())
	return file, func() {
		os.Remove(file.Name())
	}
}

func TestMakeFile(t *testing.T) {
	file, cleanup := makeFile(t, 1024)
	defer cleanup()
	file, err := os.Open(file.Name())
	require.NoError(t, err)

	bs, err := ioutil.ReadAll(file)
	require.NoError(t, err)
	require.Len(t, bs, 1024)
}

func TestChunkerImpl_Chunk(t *testing.T) {
	chunkSize := 4 * 1024
	fileMultiplier := 100

	file, cleanup := makeFile(t, fileMultiplier*chunkSize)
	defer cleanup()
	file, err := os.Open(file.Name())
	require.NoError(t, err)

	n := 0
	handler := NewFunctionHandler(func(chunk *Chunk) error {
		n++
		return nil
	})

	err = handler.HandleChunks(NewChunker(&ChunkerOpts{ChunkSize: chunkSize}).Chunk(file))
	require.NoError(t, err)
	require.Equal(t, fileMultiplier, n)
}
