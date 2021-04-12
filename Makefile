compile: bindata.go
	echo "you did it!"

bindata.go: build/tor/tor
	go-bindata -nometadata -nocompress ./build/tor

build/tor/tor:
	./build/scripts/build_tor