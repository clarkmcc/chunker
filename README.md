# Chunker
Chunker provides a simple interface for chunking data from an `io.Reader`. In addition, it provides easy-to-use utilities to chunk data directly into a gRPC stream, read the chunked data from a gRPC server, and unchunk the data back into an `io.Reader`. The following utilities are provided to make working with the chunked data easy:
* Chunker - chunks data from an `io.Reader`.
* Handler - interface for doing something with the chunked data and handling chunking errors.
* UploadFrom - A pre-built function that uses a chunker and a handler with gRPC client and server wrappers to chunk data over a gRPC stream.
* Collector - Collects chunks received over a gRPC stream and turns them into a `chunked.Group`, an `io.Reader` implementation.

## The Chunker
The chunker returns a channel where chunks will be sent, and a channel of errors. Any error will stop the chunking process.

```go
type Chunker interface {
	Chunk(r io.Reader) (<-chan *Chunk, <-chan error)
}
```

You can create a new chunker using the default options or set the options yourself.

```go
chunks, errors := chunker.NewChunker(&chunker.ChunkerOpts{
	ChunkSize: 4 * 1024, // defaults to 4kb
	BufferSize: 1024, // defaults to 1024 chunks
}).Chunk(reader)
```

## Handler
A handler is used to handle chunks received from the chunker. The default handler included with this package is a generic function handler which calls a function for each chunk and returns an error of a chunking error was encountered.

```go
err := chunker.NewFunctionHandler(func(*Chunk) error {
	// do something with the chunk
	return nil
}).HandleChunks(chunker.Chunks())
```

## Collector
A collector is used to collect the chunked data from a gRPC stream (wrapped) and assemble the chunks into a `chunked.Group`. The group ensures that chunk order is preserved, and that there are no missing chunks. The collector returns an error if any problems were detected receiving and organizing the chunks.

```go
group, err := chunk.NewCollector().Collect(WrapUploadFileServer(...))
```

## gRPC Wrappers
gRPC wrappers are used to make any gRPC stream client and server compatible with the gRPC utilities in this package (UploadFrom, Collector). 

```go
// WrappedClient knows how to wrap an RPC client and provides an interface for the 
// UploadFile function to interact with multiple rpc endpoint.
type WrappedClient interface {
	Send(chunk *protos.Chunk) error
	io.Closer
}

// WrappedServer knows how to wrap an RPC chunking server and provides an interface 
// for a chunk collector to get all the chunks needed.
type WrappedServer interface {
	Receive() (*protos.Chunk, error)
	io.Closer
}

```

The following is an example implementation of a custom gRPC client and server being wrapped:

```protobuf
service MyService {
  rpc UploadFile(stream Chunk) returns (...) {}
}
```

```go
var _ chunker.WrappedClient = &ClientWrapper{}

// ClientWrapper is a generic wrapped used to wrap rpc chunk streaming endpoint
// clients in an interface that the chunker can understand.
type ClientWrapper struct {
	SendFn  func(chunk *protos.Chunk) error
	CloseFn func() error
}

func (c *ClientWrapper) Send(chunk *protos.Chunk) error {
	return c.SendFn(chunk)
}

func (c *ClientWrapper) Close() error {
	return c.CloseFn()
}

var _ chunk.WrappedServer = &ServerWrapper{}

type ServerWrapper struct {
	ReceiveFn func() (*protos.Chunk, error)
	CloseFn   func() error
}

func (s *ServerWrapper) Receive() (*protos.Chunk, error) {
	return s.ReceiveFn()
}

func (s *ServerWrapper) Close() error {
	return s.CloseFn()
}

// Returns a wrapped client for the UploadSNMPDump client
func WrapUploadFileClient(client rpc.MyService_UploadFile) chunker.WrappedClient {
	return &ClientWrapper{
		SendFn: func(chunk *protos.Chunk) error {
			return client.Send(chunk)
		},
		CloseFn: func() error {
			_, err := client.CloseAndRecv()
			if err != nil {
				return err
			}
			return nil
		},
	}
}

func WrapUploadFileServer(srv rpc.MyService_UploadFile) chunker.WrappedServer {
	return &ServerWrapper{
		ReceiveFn: srv.Recv,
		CloseFn: func() error {
			return srv.SendAndClose(nil)
		},
	}
}
```

## Using the Wrappers
Once the wrappers are created, you can use them with the chunker on the client:

```go
err := chunk.UploadFrom(reader, WrapUploadFileClient(client))
```

and on the server:

```go
func (s *server) UploadSNMPDump(srv rpc.MyService_UploadFileServer) error {
    group, err := chunk.NewCollector().Collect(WrapUploadFileServer(srv))
    if err != nil {
        return err
    }
    b, err := ioutil.ReadAll(group.Reader())
    if err != nil {
        return err
    }
    // Do something with the de-chunked data
}
```