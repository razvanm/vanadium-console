export GOPATH := $(shell pwd)
export GOROOT := $(shell pwd)/vanadium-go-1.3
export PATH := $(GOROOT)/bin:$(PATH)

bin = app/main.nexe

.PHONY: all
all: $(bin)

$(bin): main.go vanadium-go-1.3 src
	GOOS=nacl GOARCH=amd64p32 go build -o $@ $<

vanadium-go-1.3:
	git clone --depth 1 https://vanadium.googlesource.com/release.go.ppapi \
	    vanadium-go-1.3
	cd vanadium-go-1.3/src && ./make-nacl-amd64p32.sh

src:
	git clone --depth 1 https://github.com/vanadium/core.git src/v.io
	go get github.com/shirou/gopsutil
	cd src/github.com/shirou/gopsutil && patch -p1 < ../../../../gopsutil.diff
	go get -d v.io/...

.PHONY: clean
clean:
	rm $(bin)
