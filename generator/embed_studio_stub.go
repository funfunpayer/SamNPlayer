//go:build !studio

package generator

import "fmt"

func extractStudioScripts(dir string) error {
	return nil
}

func readStudioFile(name string) ([]byte, error) {
	return nil, fmt.Errorf("not available in customer build")
}
