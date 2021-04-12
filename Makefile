torbuild = build/tor/tor
bindata = internal/bindata/bindata.go

run: $(bindata)
	go run ./

compile: $(bindata)
	echo "you did it!"

$(bindata): $(torbuild)
	go-bindata -nometadata -nocompress -o $(bindata) -pkg bindata $(torbuild)

build/tor/tor:
	./build/scripts/build_tor

clean:
	rm -f $(torbuild)
	rm -f $(bindata)