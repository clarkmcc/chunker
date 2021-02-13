.PHONY: protos

protos:
	@protoc --go_out=. protos/chunk.proto