# Build
FROM golang:alpine AS build

RUN apk add --no-cache -U git make build-base

RUN go get github.com/GeertJohan/go.rice/rice

WORKDIR /src/go-pincnic-umbrella
COPY . /src/go-pincnic-umbrella

RUN make install

# Runtime
FROM alpine:latest

COPY --from=build /go/bin/go-picocss /go-picocss

EXPOSE 8000/tcp

ENTRYPOINT ["/go-picocss"]
CMD []
