PREFIX=/usr/local
DESTDIR=${PREFIX}
BINDIR=bin
MANDIR=/usr/share/man/man1
BZ2=bzip2 -p
GO=go
INSTALL_PROGRAM=install -m 0755
INSTALL_FILE=install -m 0644

.PHONY: all install install-em uninstall uninstall-em clean

all: gomacs gomacs.1.bz2

gomacs:
	${GO} get -u -v
	${GO} build -v -o gomacs

gomacs.1.bz2: gomacs.1
	${BZ2} -f -k gomacs.1

install: gomacs gomacs.1.bz2
	${INSTALL_PROGRAM} gomacs ${DESTDIR}/${BINDIR}/gomacs
	${INSTALL_FILE} gomacs.1.bz2 ${MANDIR}/gomacs.1.bz2

install-em: gomacs gomacs.1.bz2
	${INSTALL_PROGRAM} gomacs ${DESTDIR}/${BINDIR}/em
	${INSTALL_FILE} gomacs.1.bz2 ${MANDIR}/em.1.bz2

uninstall:
	rm -rf ${DESTDIR}/${BINDIR}/gomacs
	rm -rf ${MANDIR}/gomacs.1.bz2

uninstall-em:
	rm -rf ${DESTDIR}/${BINDIR}/em
	rm -rf ${MANDIR}/em.1.bz2

clean:
	rm -rf gomacs
	rm -rf gomacs.1.bz2
