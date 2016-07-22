GOROOT_V23 := $(shell pwd)/vanadium-go-1.3
export GOPATH := $(shell pwd)
export PATH := $(GOROOT_V23)/bin:$(PATH)

bin = app/main.nexe

.PHONY: all
all: $(bin)

$(bin): main.go vanadium-go-1.3 src
	GOROOT=$(GOROOT_V23) GOOS=nacl GOARCH=amd64p32 go build -o $@ $<

vanadium-go-1.3:
	git clone --depth 1 https://vanadium.googlesource.com/release.go.ppapi \
	    vanadium-go-1.3
	cd vanadium-go-1.3/src && ./make-nacl-amd64p32.sh

src:
	git clone --depth 1 https://github.com/vanadium/core.git src/v.io
	go get -d v.io/...
	# Go 1.3 doesn't know about the vendor/ directory.
	rsync -a src/v.io/vendor/ src/

.PHONY: clean
clean:
	rm $(bin)
