DESTDIR=
prefix=/usr
sbindir=$(prefix)/sbin
datadir=$(prefix)/share
pkglibdir=$(prefix)/lib/ca-certificates
pkgdatadir=$(datadir)/ca-certificates
docdir=$(datadir)/doc/ca-certificates
mandir=$(datadir)/man
systemdsystemunitdir=$(prefix)/lib/systemd/system

all:
	$(MAKE) -C cmd/update-ca-certificates
	$(MAKE) -C plugins/certbundle
	$(MAKE) -C plugins/java
	$(MAKE) -C plugins/openssl
	$(MAKE) -C plugins/etcssl
	$(MAKE) -C plugins/nssdb

install:
	$(MAKE) -C cmd/update-ca-certificates install DESTDIR=$(DESTDIR)
	$(MAKE) -C plugins/certbundle install DESTDIR=$(DESTDIR)
	$(MAKE) -C plugins/java install DESTDIR=$(DESTDIR)
	$(MAKE) -C plugins/openssl install DESTDIR=$(DESTDIR)
	$(MAKE) -C plugins/etcssl install DESTDIR=$(DESTDIR)
	$(MAKE) -C plugins/nssdb install DESTDIR=$(DESTDIR)
	install -Dm644 COPYING -t $(DESTDIR)$(docdir)
	install -Dm644 README.md -t $(DESTDIR)$(docdir)
	install -d $(DESTDIR)$(mandir)/man8
	install -m644 doc/update-ca-certificates.8 -t $(DESTDIR)$(mandir)/man8
	install -Dm644 cmd/update-ca-certificates/ca-certificates.service -t $(DESTDIR)$(systemdsystemunitdir)
	install -Dm644 cmd/update-ca-certificates/ca-certificates-setup.service -t $(DESTDIR)$(systemdsystemunitdir)
	install -Dm644 cmd/update-ca-certificates/ca-certificates.path -t $(DESTDIR)$(systemdsystemunitdir)

tidy:
	$(MAKE) -C cmd/update-ca-certificates tidy
	$(MAKE) -C plugins/certbundle tidy
	$(MAKE) -C plugins/java tidy
	$(MAKE) -C plugins/openssl tidy
	$(MAKE) -C plugins/etcssl tidy
	$(MAKE) -C plugins/nssdb tidy

clean:
	$(MAKE) -C cmd/update-ca-certificates clean
	$(MAKE) -C plugins/certbundle clean
	$(MAKE) -C plugins/java clean
	$(MAKE) -C plugins/openssl clean
	$(MAKE) -C plugins/etcssl clean
	$(MAKE) -C plugins/nssdb clean

.PHONY: all install clean tidy
