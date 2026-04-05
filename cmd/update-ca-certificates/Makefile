DESTDIR=
prefix=/usr
sbindir=$(prefix)/sbin

TARGET=update-ca-certificates

all: $(TARGET)

$(TARGET): *.go
	pwd && \
	go mod tidy && \
	go build -v -o $@ ./

install: $(TARGET)
	install -Dm755 $(TARGET) -t $(DESTDIR)$(sbindir)

clean:
	rm -f $(TARGET)

.PHONY: all install clean
