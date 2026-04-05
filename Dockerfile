FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/foostash ./cmd/server
RUN CGO_ENABLED=0 go build -o /out/foostash-cli ./cmd/cli

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /out/foostash /usr/local/bin/foostash
COPY --from=builder /out/foostash-cli /usr/local/bin/foostash-cli
COPY --from=builder /src/migrations /migrations
ENV FOOSTASH_MIGRATIONS_DIR=/migrations
EXPOSE 8400
ENTRYPOINT ["foostash"]
CMD ["serve"]
