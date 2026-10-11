//go:build !studio

package generator

// copyStudioScripts is a no-op in the customer build (training scripts are not embedded).
func copyStudioScripts(dir string) error { return nil }
