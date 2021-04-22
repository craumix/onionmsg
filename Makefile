torbuild = build/tor/tor
bindata = internal/bindata/bindata.go
builddir = build/bin
buildbin = $(builddir)/tormsg

run: $(bindata)
	go run ./

compile: $(bindata)
	mkdir -p $(builddir)
	CGO=0 go build -ldflags="-s -w" -o $(buildbin) ./
	upx --best $(buildbin)

$(bindata): $(torbuild)
	go-bindata -nometadata -nocompress -o $(bindata) -pkg bindata $(torbuild)

$(torbuild):
	./build/scripts/build_tor

clean:
	rm -f $(torbuild)
	rm -f $(bindata)
	rm -f $(buildbin)