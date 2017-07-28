package volume

import (
	"github.com/docker/docker/volume/mountpoint"
)

// AppliedMountPointPlugin represents a mount point plugin's
// application to a specific mount point. It tracks which plugin was
// applied (both referentially and directly -- necessary for
// serialization/deserialization), what the plugin changed the mount
// point path to (if any change), and the order of mount point
// application.
type AppliedMountPointPlugin struct {
	Name       string                          // Name is the name of the plugin (for later lookup)
	plugin     *mountpoint.Plugin              // plugin stores the plugin object
	Attachment mountpoint.MountPointAttachment // Attachment contains whatever changes the plugin has made to the mount
	Clock      int                             // Clock is a positive integer used to ensure mount detachments occur in the correct order
}

// Plugin will retrieve the Plugin object or create a new one if none is available
func (p AppliedMountPointPlugin) Plugin() (*mountpoint.Plugin, error) {
	if p.plugin == nil {
		plugin, err := mountpoint.NewMountPointPlugin(p.Name)
		if err != nil {
			return nil, err
		}
		p.plugin = &plugin
	}
	return p.plugin, nil
}

// EffectiveSource is the directory to use for a mount even after plugins may
// have changed the original source directory
func (m *MountPoint) EffectiveSource() string {
	for i := len(m.AppliedPlugins) - 1; i >= 0; i-- {
		appliedPlugin := m.AppliedPlugins[i]
		if appliedPlugin.Attachment.EffectiveSource != "" {
			return appliedPlugin.Attachment.EffectiveSource
		}
	}
	return m.Source
}

// PushPlugin pushes a new applied plugin onto the mount point's plugin stack
func (m *MountPoint) PushPlugin(plugin mountpoint.Plugin, attachment mountpoint.MountPointAttachment, clock int) {
	appliedPlugin := AppliedMountPointPlugin{
		Name:       plugin.Name(),
		plugin:     &plugin,
		Attachment: attachment,
		Clock:      clock,
	}
	m.AppliedPlugins = append(m.AppliedPlugins, appliedPlugin)
}

// PopPlugin removes and returns a plugin from the mount point's plugin stack
func (m *MountPoint) PopPlugin() *AppliedMountPointPlugin {
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
