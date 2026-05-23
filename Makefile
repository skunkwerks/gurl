GO?=		go
GOFLAGS?=
PREFIX?=	/usr/local
BINDIR?=	${PREFIX}/bin

PROG=		gurl
MODULE=		github.com/skunkwerks/gurl

all: test build

build: ${SRCS}
	${GO} build ${GOFLAGS} -ldflags "-s -X ${MODULE}/version.Version=$$(git describe --tags --always --dirty)" -o ${PROG}

test: fmt vet
	${GO} test -count=1 ./...

fmt:
	${GO} fmt ./...

vet:
	${GO} vet ./...

clean:
	rm -f ${PROG}

.PHONY: all build clean fmt test vet
