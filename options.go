package iavl

// Options define tree options.
type Options struct {
	// Sync synchronously flushes all writes to storage, using e.g. the fsync syscall.
	// Disabling this significantly improves performance, but can lose data on e.g. power loss.
	Sync bool

	// InitialVersion specifies the initial version number. If any versions lower than this exists,
	// an error will be returned. Only used for the initial SaveVersion() call.
	InitialVersion uint64
}

// DefaultOptions returns the default options for IAVL.
func DefaultOptions() *Options {
	return &Options{}
}
