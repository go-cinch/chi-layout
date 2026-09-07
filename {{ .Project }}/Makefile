APP := {{ .Computed.service_name_kebab }}
MODULE := $(shell go list -m)
VERSION ?= $(shell git describe --tags --always 2>/dev/null)
ifeq ($(strip $(VERSION)),)
VERSION := dev
endif
LDFLAGS := -s -w -X $(MODULE)/internal/common/config.Version=$(VERSION)

.PHONY: config api gen tidy fmt lint test build run clean

config:
	go run -mod=mod ./internal/cmd/configgen -config-dir ./conf -output ./internal/common/config/config.gen.go

api:
	go run ./internal/cmd/modulegen -modules-dir ./internal/modules -app-file ./internal/app/app.go -go-mod ./go.mod -output ./internal/app/modules.gen.go
	go run ./internal/cmd/docsgen -source ./internal -config-dir ./conf -output ./internal/docs/openapi.yaml -title "$(APP) API"

gen: config api

tidy:
	go mod tidy

fmt:
	gofmt -w .

lint: gen
	go vet ./...
	@test -z "$$(gofmt -l .)" || (gofmt -l . && exit 1)

test: gen
	go test -race -coverprofile=coverage.out ./...
	awk -f ./scripts/check-coverage.awk coverage.out

build: gen
	mkdir -p bin
	go build -trimpath -ldflags "$(LDFLAGS)" -o ./bin/$(APP) ./cmd/server

run: gen
	go run ./cmd/server -c ./conf

clean:
	rm -rf ./bin ./coverage.out
