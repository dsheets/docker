package mountpoint

import "github.com/docker/docker/volume"

// MountPoint is the intersection point between a mount and a
// container. It specifies which volume is to be used and where inside
// a container it should be mounted. mountpoint.MountPoint is simply
// volume.MountPoint with an additional AppliedPlugins field that
// tracks a stack of mount point plugins participating in the mount.
type MountPoint struct {
	*volume.MountPoint
	// AppliedPlugins is the stack of mount point plugins that have been applied to this mount
	AppliedPlugins []AppliedPlugin
}

// AppliedPlugin represents a mount point plugin's application to a
// specific mount point. It tracks which plugin was applied (both
// referentially and directly -- necessary for
// serialization/deserialization), what the plugin changed the mount
// point path to (if any change), and the order of mount point
// application.
type AppliedPlugin struct {
	Name      string  // Name is the name of the plugin (for later lookup)
	plugin    *Plugin `json:"-"` // plugin stores the plugin object
	MountPath string  // MountPath is the new path of the mount (or "" if unchanged)
	Clock     int     // Clock is a positive integer used to ensure mount detachments occur in the correct order
}

// Plugin will retrieve the Plugin object or create a new one if none is available
func (p AppliedPlugin) Plugin() (*Plugin, error) {
	if p.plugin == nil {
		plugin, err := newMountPointPlugin(p.Name)
		if err != nil {
			return nil, err
		}
		p.plugin = &plugin
	}
	return p.plugin, nil
}

// Source is the directory to use for a mount even after plugins may
// have changed the original source directory
func (m *MountPoint) Source() string {
	for i := len(m.AppliedPlugins) - 1; i >= 0; i-- {
		appliedPlugin := m.AppliedPlugins[i]
		if appliedPlugin.MountPath != "" {
			return appliedPlugin.MountPath
		}
	}
	return m.MountPoint.Source
}

// PushPlugin pushes a new applied plugin onto the mount point's plugin stack
func (m *MountPoint) PushPlugin(plugin Plugin, newMountPoint string, clock int) {
	appliedPlugin := AppliedPlugin{
		Name:      plugin.Name(),
		plugin:    &plugin,
		MountPath: newMountPoint,
		Clock:     clock,
	}
	m.AppliedPlugins = append(m.AppliedPlugins, appliedPlugin)
}

// PopPlugin removes and returns a plugin from the mount point's plugin stack
func (m *MountPoint) PopPlugin() *AppliedPlugin {
	stack := m.AppliedPlugins
	if len(stack) > 0 {
		plugin := &stack[len(stack)-1]
		m.AppliedPlugins = stack[0 : len(stack)-1]
		return plugin
	}
	return nil
}

// TopClock returns the Clock value from the plugin on the top of the
// mount point's plugin stack or 0 if the stack is empty
func (m *MountPoint) TopClock() int {
	stackSize := len(m.AppliedPlugins)
	if stackSize > 0 {
		return m.AppliedPlugins[stackSize-1].Clock
	}
	return 0
}
