PREFIX=/usr/local
DESTDIR=${PREFIX}
BINDIR=bin
MANDIR=/usr/share/man/man1
BZ2=bzip2
GO=go
INSTALL_PROGRAM=install -m 0755
INSTALL_FILE=install -m 0644
VERSION=git
ARCH=$(shell arch)

.PHONY: all dist install install-em uninstall uninstall-em clean

all: gomacs gomacs.1.bz2

gomacs:
	${GO} get -u -v
	${GO} build -v -o gomacs

gomacs.1.bz2: gomacs.1
	${BZ2} -f -k gomacs.1

gomacs.1: gomacs
	sed -n -e '1,/^<<BINDINGS>>$$/p' gomacs.1.in | sed -e '$$ d' > gomacs.1
	./gomacs -D | sed -e "s/^ \(.*\) - \(.*\)/.It \1\n\2/" -e's/\\/\\e/g' -e 's/)/\\\&)/g' >> gomacs.1
	sed -e '1,/^<<BINDINGS>>$$/d' gomacs.1.in >> gomacs.1

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
	rm -rf dist

.ONESHELL:
dist: gomacs gomacs.1.bz2
	mkdir -pv dist/gomacs_${ARCH}-${VERSION}
	cd dist/gomacs_${ARCH}-${VERSION}
	cp ../../gomacs ../../gomacs.1.bz2 ../../gomacs.png ../../LICENSE \
	../../README.md ../../CHANGELOG ./
	cp --recursive ../../examples ./
	cd ../
	tar cvzf gomacs_${ARCH}-${VERSION}.tgz gomacs_${ARCH}-${VERSION}
