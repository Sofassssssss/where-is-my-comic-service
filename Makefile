container_runtime := $(shell which docker || which podman)

$(info using ${container_runtime})

up: down
	${container_runtime} compose up --build -d

down:
	${container_runtime} compose down

clean:
	${container_runtime} compose down -v

run-tests: 
	${container_runtime} run --rm --network=host tests:latest

test:
	make clean
	make up
	@echo wait cluster to start && sleep 10
	make run-tests
	make clean
	@echo "test finished"

lint:
	make -C search-services lint

proto:
	make -C search-services protobuf

unit:
	cd search-services && \
	go test ./... -coverprofile=cover.out ; \
	grep -v -E "(/config/|/proto/|/closers/|main\.go|error\.go|mapper\.go|subscriber\.go|concurrency\.go|metric\.go|rate\.go|/adapters/db/|/adapters/xkcd/|/adapters/words/|/api/adapters/(isearch|search|update|aaa|words)/|/update/adapters/nats/)" cover.out > cover_filtered.out ; \
	go tool cover -html=cover_filtered.out -o cover.html && \
	mv cover.html ..

tools:
	go install github.com/yoheimuta/protolint/cmd/protolint@v0.56.3
	go install golang.org/x/tools/cmd/goimports@v0.31.0
	go install github.com/fullstorydev/grpcurl/cmd/grpcurl@v1.9.3
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.5
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8

	@echo "checking protobuf compiler"
	@command -v protoc >/dev/null 2>&1 && echo OK || exit 1

