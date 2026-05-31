DESTDIR=
prefix=/usr
libdir=$(prefix)/lib
plugindir=$(libdir)/ca-certificates/update.d

TARGET=nssdb.plugin

all: $(TARGET)

$(TARGET): *.go
	pwd && \
	go build -v -o $@ ./

install: $(TARGET)
	install -Dm755 $(TARGET) -t $(DESTDIR)$(plugindir)

tidy:
	go mod tidy

clean:
	rm -f $(TARGET)

.PHONY: all install clean tidy
