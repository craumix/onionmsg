// +build !internalTor

package tor

func torBinaryPath() (string, error) {
	return "tor", nil
}
