//+build !test

package mountpoint

import (
	"strings"
)

// Middleware interposes local file system mount points
type Middleware interface {
	// Name returns the registered middleware name. Plugin names have
	// a 'plugin:' prefix
	Name() string

	// PluginName returns the name of the plugin implementing this
	// mount point middleware (if any). If the middleware is not
	// plugin-based, PluginName returns the empty string.
	PluginName() string

	// Patterns returns the mount point patterns that this plugin interposes
	Patterns() []MountPointPattern

	// MountPointProperties returns the properties of the mount point plugin
	MountPointProperties(*PropertiesRequest) (*PropertiesResponse, error)

	// MountPointAttach prepares one or more mount points for a container
	MountPointAttach(*AttachRequest) (*AttachResponse, error)

	// MountPointDetach releases one or more mount points from a container
	MountPointDetach(*DetachRequest) (*DetachResponse, error)
}

func PluginNameOfMiddlewareName(middlewareName string) string {
	pluginPrefix := "plugin:"
	if strings.HasPrefix(middlewareName, pluginPrefix) {
		return strings.TrimPrefix(middlewareName, pluginPrefix)
	} else {
		return ""
	}
}
