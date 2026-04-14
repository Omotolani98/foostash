# syntax=docker/dockerfile:1.7

FROM golang:1.25.1-alpine AS builder
WORKDIR /src

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown
ENV CGO_ENABLED=0 GOOS=linux

RUN go build \
    -ldflags="-s -w \
      -X github.com/Omotolani98/foostash/cli.Version=${VERSION} \
      -X github.com/Omotolani98/foostash/cli.Commit=${COMMIT} \
      -X github.com/Omotolani98/foostash/cli.Date=${DATE}" \
    -o /out/foostash ./cmd/foostash

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/foostash /usr/local/bin/foostash
EXPOSE 8400
ENTRYPOINT ["/usr/local/bin/foostash", "serve"]
