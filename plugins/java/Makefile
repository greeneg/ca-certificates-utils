DESTDIR=
prefix=/usr
libdir=$(prefix)/lib
plugindir=$(libdir)/ca-certificates/update.d

TARGET=java.plugin

all: $(TARGET)

$(TARGET): all
	pwd && \
	go mod tidy && \
	go build -v -o $@ ./

install: $(TARGET)
	install -Dm755 $(TARGET) -t $(DESTDIR)$(plugindir)

clean:
	rm -f $(TARGET)

.PHONY: all install clean
