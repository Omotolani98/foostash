FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/foostash-server ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=build /bin/foostash-server /usr/local/bin/foostash-server
COPY migrations /migrations
EXPOSE 8080
ENV FOOSTASH_PORT=8080
ENV FOOSTASH_MIGRATIONS_DIR=/migrations
ENTRYPOINT ["foostash-server"]
CMD ["serve"]
