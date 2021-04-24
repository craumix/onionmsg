// +build internalTor

//go:generate go-bindata -nometadata -nocompress -tags internalTor -o ./internal/tor/bindata.go -pkg tor tor

package main
