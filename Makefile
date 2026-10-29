GO?=		go
GOFLAGS?=
PREFIX?=	/usr/local
BINDIR?=	${PREFIX}/bin
MANDIR?=	${PREFIX}/share/man

PROG=		gurl
MODULE=		github.com/skunkwerks/gurl

all: test build

# The version comes from git when available. In the CI firecracker VM the
# checkout has no .git, so fall back to the baked-in version.Version.
VERSION_SRC=$$(git describe --tags --always --dirty 2>/dev/null || echo "dev")

build: ${SRCS}
	${GO} build -buildvcs=false ${GOFLAGS} -ldflags "-s -X ${MODULE}/version.Version=${VERSION_SRC}" -o ${PROG}

# The manual page is emitted by the binary itself, so the shipped page and
# the running program can never drift. Run `mandoc -Tlint gurl.1` to lint.
man: gurl.1

gurl.1: ${PROG}
	./${PROG} -man > $@

install: build install-man
	${INSTALL_PROGRAM} ${PROG} ${DESTDIR}${BINDIR}/${PROG}

install-man: gurl.1
	mkdir -p ${DESTDIR}${MANDIR}/man1
	${INSTALL_MAN} gurl.1 ${DESTDIR}${MANDIR}/man1/gurl.1

test: fmt vet
	${GO} test -count=1 ./...

fmt:
	${GO} fmt ./...

vet:
	${GO} vet ./...

clean:
	rm -f ${PROG} gurl.1

.PHONY: all build clean fmt install install-man man test vet
