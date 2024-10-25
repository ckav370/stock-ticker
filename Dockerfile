# hadolint global ignore=DL3018
FROM golang:1.22-alpine3.18 AS build

RUN apk update && apk add --no-cache gcc g++ openssl libc-dev librdkafka-dev pkgconf ca-certificates curl
ENV GOOS=linux
ENV GOARCH=amd64

ENV SRC_HOME=/go/src/github.com/ckav370/stock-ticker

WORKDIR $SRC_HOME

COPY . $SRC_HOME

RUN go mod download && go build -tags musl -o bin/stock-ticker main.go

# Final stage
FROM alpine:latest
ENV SRC_HOME=/go/src/github.com/ckav370/stock-ticker

# Copy the built binary from the build stage
COPY --from=build $SRC_HOME/bin/stock-ticker /usr/local/bin/

# Command to run the executable
CMD ["stock-ticker"]
