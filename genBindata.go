// +build internalTor

//go:generate go-bindata -nometadata -nocompress -tags internalTor -o ./internal/tor/bindata.go -pkg tor ./third_party/tor/tor

package main
