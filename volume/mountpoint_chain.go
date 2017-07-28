package volume

import (
	"fmt"
	"os"
	"sync"

	mounttypes "github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/pkg/plugingetter"
	"github.com/docker/docker/volume/mountpoint"
	"github.com/pkg/errors"
)

// MountPointChain uses a list of plugins to interpose on mount point
// attachment and detachment
type MountPointChain struct {
	mu      sync.Mutex
	plugins []mountpoint.Plugin
}

// NewMountPointChain creates a new Chain with a slice of plugin names.
func NewMountPointChain(names []string, pg plugingetter.PluginGetter) (*MountPointChain, error) {
	mountpoint.SetPluginGetter(pg)
	plugins, err := mountpoint.NewPlugins(names)
	if err != nil {
		return nil, err
	}
	return &MountPointChain{
		plugins: plugins,
	}, nil
}

// AttachMounts runs a list of mount attachments through a mount point plugin chain
func (c *MountPointChain) AttachMounts(id string, mounts []*MountPoint) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var mountPointClock int

	for _, plugin := range c.plugins {
		var selectedMounts []*MountPoint
		patterns := plugin.Patterns()

		mountPointClock++

		// select mounts for this plugin
		for _, mount := range mounts {
			for _, pattern := range patterns {
				if mountpoint.PatternMatches(pattern, pluginMountPointOfMountPoint(mount)) {
					selectedMounts = append(selectedMounts, mount)
					break
				}
			}
		}

		if len(selectedMounts) > 0 {
			// send attachment request to the plugin
			var pmounts []*mountpoint.MountPoint
			for _, selectedMount := range selectedMounts {
				pmounts = append(pmounts, pluginMountPointOfMountPoint(selectedMount))
			}
			request := &mountpoint.AttachRequest{id, pmounts}
			response, err := plugin.MountPointAttach(request)
			if err != nil {
				return c.unwindAttachOnErr(plugin.Name(), id, mounts, err)
			}
			if !response.Success {
				return c.unwindAttachOnErr(plugin.Name(), id, mounts, errors.New(response.Err))
			}

			// annotate the mount points with the applied plugin
			for k, attachment := range response.Attachments {
				if k >= len(selectedMounts) {
					break
				}
				if attachment.Attach {
					selectedMounts[k].PushPlugin(plugin, attachment.MountPoint, mountPointClock)
				}
			}
		}
	}

	return nil
}

// DetachMounts detaches mounts from a mount point plugin chain
func (c *MountPointChain) DetachMounts(container string, mounts map[string]*MountPoint) error {
	var list []*MountPoint
	for _, mp := range mounts {
		list = append(list, mp)
	}
	return unwind(container, list)
}

// unwindAttachOnErr will clean up previous attachments if an error
// occurs during attachment
func (c *MountPointChain) unwindAttachOnErr(pluginName, container string, mounts []*MountPoint, err error) (ret error) {
	defer func() {
		ret = errors.Wrap(ret, "plugin "+pluginName+" failed with error")
	}()

	e := unwind(container, mounts)
	if e != nil {
		return errors.Wrap(err, fmt.Sprintf("%s", e))
	}

	ret = err
	return ret
}

// unwind is used to detach all plugins participating in a container's
// mount points
func unwind(container string, mounts []*MountPoint) error {
	var err error
	var plugin *mountpoint.Plugin
	moreToUnwind := true

	for moreToUnwind {
		maxClock := 0
		moreToUnwind = false
		for _, mount := range mounts {
			maxClock = max(mount.TopClock(), maxClock)
		}

		if maxClock > 0 {
			moreToUnwind = true
			for _, mount := range mounts {
				if mount.TopClock() < maxClock {
					continue
				}

				appliedPlugin := mount.PopPlugin()
				if appliedPlugin != nil {
					if plugin == nil {
						p, e := appliedPlugin.Plugin()
						if e != nil {
							errString := fmt.Sprintf("unwind plugin retrieval error: \"%s\"", e)
							return stackError(err, errString)
						}
						plugin = p
					} else if (*plugin).Name() != appliedPlugin.Name {
						return fmt.Errorf("plugin inconsistency %s != %s", (*plugin).Name(), appliedPlugin.Name)
					}
				}
			}
			request := &mountpoint.DetachRequest{container}
			response, e := (*plugin).MountPointDetach(request)
			if e != nil {
				errString := fmt.Sprintf("unwind detach API error for %s: \"%s\"", (*plugin).Name(), e)
				return stackError(err, errString)
			}
			if !response.Success {
				errString := fmt.Sprintf("unwind detach plugin %s error: \"%s\"", (*plugin).Name(), response.Err)
				err = stackError(err, errString)
				if !response.Recoverable {
					return err
				}
			}
		}
		plugin = nil
	}
	return err
}

// stackError will wrap err in errString if err is an error or create
// a new error from errString if err is nil
func stackError(err error, errString string) error {
	if err == nil {
		return errors.New(errString)
	}
	return errors.Wrap(err, errString)
}

// max is the greater integer of a and b. Seriously?
func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}

// SetPlugins sets the mount point plugins in the chain
func (c *MountPointChain) SetPlugins(names []string) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.plugins, err = mountpoint.NewPlugins(names); err != nil {
		return err
	}
	return nil
}

// DisableMountPointPlugin removes the mount point plugin from the chain
func (c *MountPointChain) DisableMountPointPlugin(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// There may still be mounts which are relying on it during tear
	// down
	var plugins []mountpoint.Plugin
	for _, plugin := range c.plugins {
		if plugin.Name() != name {
			plugins = append(plugins, plugin)
		}
	}
	c.plugins = plugins
}

// EnableMountPointPlugin appends a mount point plugin to the chain
func (c *MountPointChain) EnableMountPointPlugin(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	plugin, err := mountpoint.NewMountPointPlugin(name)
	if err != nil {
		return err
	}
	c.plugins = append(c.plugins, plugin)
	return nil
}

func mountPointTypeOfAPIType(t mounttypes.Type) mountpoint.Type {
	var typ mountpoint.Type
	switch t {
	case mounttypes.TypeBind:
		typ = mountpoint.TypeBind
	case mounttypes.TypeVolume:
		typ = mountpoint.TypeVolume
	case mounttypes.TypeTmpfs:
		typ = mountpoint.TypeTmpfs
	}
	return typ
}

func pluginMountPointOfMountPoint(mp *MountPoint) *mountpoint.MountPoint {
	typ := mountPointTypeOfAPIType(mp.Type)
	var labels map[string]string
	var driverOptions map[string]string
	if mp.Spec.VolumeOptions != nil {
		labels = mp.Spec.VolumeOptions.Labels
		driverOptions = mp.Spec.VolumeOptions.DriverConfig.Options
	}
	var scope mountpoint.Scope
	if v, ok := mp.Volume.(DetailedVolume); ok {
		scope = mountpoint.Scope(v.Scope())
	}
	var sizeBytes int64
	var mode os.FileMode
	if mp.Spec.TmpfsOptions != nil {
		sizeBytes = mp.Spec.TmpfsOptions.SizeBytes
		mode = mp.Spec.TmpfsOptions.Mode
	}
	return &mountpoint.MountPoint{
		Source:          mp.Source,
		EffectiveSource: mp.EffectiveSource(),
		Destination:     mp.Destination,
		ReadOnly:        !mp.RW,
		Name:            mp.Name,
		Driver:          mp.Driver,
		Type:            typ,
		Mode:            mp.Mode,
		Propagation:     mp.Propagation,
		ID:              mp.ID,
		Consistency:     mp.Spec.Consistency,
		Labels:          labels,
		DriverOptions:   driverOptions,
		Scope:           scope,
		SizeBytes:       sizeBytes,
		MountMode:       mode,
		AppliedPlugins:  pluginAppliedPluginsOfAppliedPlugins(mp.AppliedPlugins),
	}
}

func pluginAppliedPluginsOfAppliedPlugins(plugins []AppliedMountPointPlugin) (ps []mountpoint.AppliedPlugin) {
	for _, p := range plugins {
		ps = append(ps, mountpoint.AppliedPlugin{
			Name:       p.Name,
			MountPoint: p.Attachment,
		})
	}

	return ps
}
