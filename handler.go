package chunk

// ChunkHandler knows how to handle a chunk received from a chunker
type Handler interface {
	HandleChunks(<-chan *Chunk, <-chan error) error
}

// ChunkFunc knows how to do something with a chunk, and returns an error if
// that something fails.
type ChunkFunc func(*Chunk) error

var _ Handler = &FunctionHandler{}

type FunctionHandler struct {
	fn ChunkFunc
}

func (f *FunctionHandler) HandleChunks(chunks <-chan *Chunk, errors <-chan error) error {
loop:
	for {
		select {
		case chunk, ok := <-chunks:
			if !ok {
				break loop
			}
			err := f.fn(chunk)
			if err != nil {
				return err
			}
		case err := <-errors:
			return err
		}
	}
	return nil
}

func NewFunctionHandler(fn ChunkFunc) *FunctionHandler {
	return &FunctionHandler{fn: fn}
}
