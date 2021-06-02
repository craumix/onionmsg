// +build !internalTor

package tor

func getExePath() (string, error) {
	return "tor", nil
}
