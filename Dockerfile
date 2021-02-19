FROM golang:1.15

ADD . /go/src/github.com/skunkwerks/gurl

RUN go install github.com/skunkwerks/gurl

ENTRYPOINT ["/go/bin/gurl"]
