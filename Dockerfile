# Build
FROM golang:alpine AS build

RUN apk add --no-cache -U git make build-base
Dockerfile
RUN go get github.com/GeertJohan/go.rice/rice

WORKDIR /src/go-picocss
COPY . /src/go-picocss

RUN make install

# Runtime
FROM alpine:latest

COPY --from=build /go/bin/go-picocss /go-picocss

EXPOSE 8000/tcp

ENTRYPOINT ["/go-picocss"]
CMD []
