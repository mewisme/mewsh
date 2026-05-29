//go:build !windows

package terminal

// DetachFromConsole is a no-op on Unix; Setsid on spawned children is sufficient.
func DetachFromConsole() {}
