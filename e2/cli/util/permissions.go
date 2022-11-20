package util

import (
	"io/fs"
)

// These constants are meant to be used as reasonable default values for files and directories created by Subo.
// nolint:godot
const (
	PermDirectory         fs.FileMode = 0755 // rwxr-xr-x
	PermDirectoryPrivate  fs.FileMode = 0700 // rwx------
	PermExecutable        fs.FileMode = 0755 // rwxr-xr-x
	PermExecutablePrivate fs.FileMode = 0700 // rwx------
	PermFile              fs.FileMode = 0644 // rw-r--r--
	PermFilePrivate       fs.FileMode = 0600 // rw-------
)
