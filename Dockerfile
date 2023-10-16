FROM golang:1.21-alpine as build

ENV CGO_ENABLED=0

WORKDIR /app
COPY go.* ./
COPY cmd cmd
COPY internal internal
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
        go build \
            -ldflags="-w -s" \
            -o multiproxy \
            cmd/multiproxy/main.go

FROM alpine

COPY --from=build /app/multiproxy /multiproxy

CMD ["/multiproxy"]
