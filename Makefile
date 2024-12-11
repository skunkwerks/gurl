GO?=		go
GOFLAGS?=
PREFIX?=	/usr/local
BINDIR?=	${PREFIX}/bin

PROG=		gurl

all: test build

build: ${SRCS}
	${GO} build ${GOFLAGS} -o ${PROG}

test:
	${GO} test -v ./...

clean:
	rm -f ${PROG}

.PHONY: all clean test
