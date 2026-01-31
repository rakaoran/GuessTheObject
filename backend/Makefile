.PHONY: protogen test

protogen:
	protoc --go_out=. --go_opt=paths=source_relative domain/protobuf/*.proto
test:
	go test ./...