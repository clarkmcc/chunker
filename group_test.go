package chunk

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"testing"
)

func o(n int) *Chunk {
	return &Chunk{o: int64(n)}
}

func toChunks(t *testing.T, size int, data []byte) *Group {
	group := &Group{}
	err := chunkAndDo(bytes.NewBuffer(data), size, func(chunk *Chunk) {
		group.Add(chunk)
	})
	require.NoError(t, err)
	return group
}

func TestGroup_MissingChunks(t *testing.T) {
	group := Group{o(0), o(1), o(2), o(3), o(4), o(5)}
	require.False(t, group.MissingChunks())
	group = Group{o(0), o(1), o(2), o(3), o(5), o(6)}
	require.True(t, group.MissingChunks())
}

func TestGroup_Bytes(t *testing.T) {
	// should be 6 bytes
	data := []byte("foobar")
	group := toChunks(t, 2, data)
	require.Equal(t, 3, group.Len())
	require.Equal(t, "foobar", string(group.Bytes()))
}

func TestGroup_Read(t *testing.T) {
	data := []byte("foobar")
	group := toChunks(t, 2, data)
	require.Equal(t, 3, group.Len())
	bs, err := ioutil.ReadAll(group.Reader())
	require.NoError(t, err)
	require.Equal(t, "foobar", string(bs))
}
