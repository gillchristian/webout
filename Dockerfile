FROM golang:1.11.1

WORKDIR /go/src/github.com/gillchristian/webout

COPY go.mod go.mod
COPY go.sum go.sum

ENV GO111MODULE on
ENV GOOS linux
ENV GOARCH amd64
ENV CGO_ENABLED 0

RUN go mod download

COPY . .

WORKDIR /go/src/github.com/gillchristian/webout/server

# TODO: make `go run` work for development

RUN go build

RUN chmod +x server

EXPOSE 8989

CMD ["./server", "--port=8989"]
