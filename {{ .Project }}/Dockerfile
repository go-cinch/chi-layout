FROM golang:1.27.1-alpine3.23 AS builder

ENV CGO_ENABLED=0
ARG VERSION=dev

RUN apk add --no-cache ca-certificates git

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN module=$(go list -m) && \
    go build -trimpath -ldflags "-s -w -X ${module}/internal/common/config.Version=${VERSION}" -o /out/{{ .Computed.service_name_kebab }} ./cmd/server

FROM alpine:3.23

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S -G app -u 10001 app

WORKDIR /app

COPY --from=builder /out/{{ .Computed.service_name_kebab }} /app/{{ .Computed.service_name_kebab }}
COPY conf /app/conf

USER app

EXPOSE {{ .Scaffold.http_port }}

ENTRYPOINT ["/app/{{ .Computed.service_name_kebab }}"]
CMD ["-c", "/app/conf"]
