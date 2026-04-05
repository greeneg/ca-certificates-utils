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
	$(MAKE) -C cmd/update-ca-certificates all
	$(MAKE) -C plugins/certbundle all
	$(MAKE) -C plugins/java all
	$(MAKE) -C plugins/openssl all
	$(MAKE) -C plugins/etcssl all

install:
	$(MAKE) -C cmd/update-ca-certificates install DESTDIR=$(DESTDIR)
	$(MAKE) -C plugins/certbundle install DESTDIR=$(DESTDIR)
	$(MAKE) -C plugins/java install DESTDIR=$(DESTDIR)
	$(MAKE) -C plugins/openssl install DESTDIR=$(DESTDIR)
	$(MAKE) -C plugins/etcssl install DESTDIR=$(DESTDIR)
	install -Dm644 COPYING -t $(DESTDIR)$(docdir)
	install -Dm644 README.md -t $(DESTDIR)$(docdir)
	install -d $(DESTDIR)$(mandir)/man8
	install -m644 doc/update-ca-certificates.8 -t $(DESTDIR)$(mandir)/man8
	install -Dm644 cmd/update-ca-certificates/ca-certificates.service -t $(DESTDIR)$(systemdsystemunitdir)
	install -Dm644 cmd/update-ca-certificates/ca-certificates-setup.service -t $(DESTDIR)$(systemdsystemunitdir)
	install -Dm644 cmd/update-ca-certificates/ca-certificates.path -t $(DESTDIR)$(systemdsystemunitdir)

clean:
	$(MAKE) -C cmd/update-ca-certificates clean
	$(MAKE) -C plugins/certbundle clean
	$(MAKE) -C plugins/java clean
	$(MAKE) -C plugins/openssl clean
	$(MAKE) -C plugins/etcssl clean

.PHONY: all install clean
