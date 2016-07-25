GOROOT_V23 := $(shell pwd)/vanadium-go-1.3
export GOPATH := $(shell pwd)
export PATH := $(GOROOT_V23)/bin:$(PATH)

bin = app/main.nexe
js = app/background.js app/main.js app/page.js

.PHONY: all
all: $(bin) $(js)

$(bin): main.go vanadium-go-1.3 src
	GOROOT=$(GOROOT_V23) GOOS=nacl GOARCH=amd64p32 go build -o $@ $<

vanadium-go-1.3:
	git clone --depth 1 https://github.com/razvanm/vanadium-go-1.3.git
	cd vanadium-go-1.3/src && ./make-nacl-amd64p32.sh

src:
	git clone --depth 1 https://github.com/vanadium/core.git src/v.io
	GOROOT=$(GOROOT_V23) go get -d v.io/...
	# Go 1.3 doesn't know about the vendor/ directory.
	rsync -a src/v.io/vendor/ src/

node_modules:
	npm install

app/%.js: javascript/%.js node_modules
	$(shell npm bin)/browserify -o $@ $<

.PHONY: clean
clean:
	rm -f $(bin) $(js)

.PHONY: distclean
distclean: clean
	rm -rf src node_modules vanadium-go-1.3
