//go:build !windows

package executor

func normalizePlatformOutput(data []byte) string { return normalizePortableOutput(data) }
