FROM golang:1.23-alpine AS build

RUN apk update && apk add --no-cache gcc g++ openssl libc-dev librdkafka-dev pkgconf ca-certificates curl
ENV GOOS=linux
ENV GOARCH=amd64

ENV SRC_HOME=/go/src/github.com/ckav370/stock-ticker

WORKDIR $SRC_HOME

COPY . $SRC_HOME

RUN go mod download && go build -tags musl -o bin/stock-ticker main.go

FROM gcr.io/distroless/static-debian12
ENV SRC_HOME=/go/src/github.com/ckav370/stock-ticker

COPY --from=build $SRC_HOME/bin/stock-ticker /usr/local/bin/

CMD ["stock-ticker"]