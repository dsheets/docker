package mountpoint

import (
	"fmt"
	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
	mounttypes "github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/pkg/plugingetter"
	"github.com/docker/docker/volume"
	"github.com/pkg/errors"
)

// Chain uses a list of plugins to interpose on mount point attachment
// and detachment
type Chain struct {
	mu      sync.Mutex
	plugins []Plugin
}

// NewChain creates a new Chain with a slice of plugin names.
func NewChain(names []string, pg plugingetter.PluginGetter) (*Chain, error) {
	SetPluginGetter(pg)
	plugins, err := newPlugins(names)
	if err != nil {
		return nil, err
	}
	return &Chain{
		plugins: plugins,
	}, nil
}

// AttachMounts runs a list of mount attachments through a mount point plugin chain
func (c *Chain) AttachMounts(id string, mounts []*MountPoint) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var mountPointClock int

	for _, plugin := range c.plugins {
		var selectedMounts []*MountPoint
		types := plugin.Types()
		volumePatterns := plugin.VolumePatterns()

		mountPointClock++

		// select mounts for this plugin
		for _, mount := range mounts {
			if types[mount.MountPoint.Type] {
				switch mount.MountPoint.Type {
				case mounttypes.TypeBind:
					selectedMounts = append(selectedMounts, mount)
				case mounttypes.TypeVolume:
					if len(volumePatterns) == 0 || doVolumePatternsMatch(plugin.VolumePatterns(), mount) {
						selectedMounts = append(selectedMounts, mount)
					}
				default: // only bind and volume mounts are supported right now
				}
			}
		}

		if len(selectedMounts) > 0 {
			// send attachment request to the plugin
			var pmounts []*MountPoint
			for _, selectedMount := range selectedMounts {
				pmounts = append(pmounts, selectedMount)
			}
			request := &AttachRequest{id, pmounts}
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
					selectedMounts[k].PushPlugin(plugin, attachment.NewMountPoint, mountPointClock)
				}
			}
		}
	}

	return nil
}

// doVolumePatternsMatch checks if any pattern matches a mount
// point. If no patterns are supplied, the mount point match
// conservatively succeeds.
func doVolumePatternsMatch(volumePatterns []VolumePattern, mount *MountPoint) bool {
	volume := mount.MountPoint.Volume
	volumeDriver := volume.DriverName()

	if len(volumePatterns) == 0 {
		return true
	}

	for _, pattern := range volumePatterns {
		if volumeDriver == pattern.VolumePlugin && doesOptionPatternMatch(pattern.OptionPattern, volume) {
			return true
		}
	}

	return false
}

// doesOptionPatternMatch checks
func doesOptionPatternMatch(pattern OptionPattern, vol volume.Volume) bool {
	if v, ok := vol.(volume.DetailedVolume); ok {
		options := v.Options()

		for keyPattern, patternValueSet := range pattern {
			keyLiteral := parsePattern(keyPattern)
			if optsValue, ok := options[keyLiteral]; ok {
				if isNegation(keyPattern) {
					return false
				}
				optsValueSet := strings.Split(optsValue, ",")
				for _, segmentPattern := range patternValueSet {
					segmentLiteral := parsePattern(segmentPattern)
					negate := isNegation(segmentPattern)
					if negate == elementOf(segmentLiteral, optsValueSet) {
						return false
					}
				}
			} else {
				if !isNegation(keyPattern) {
					return false
				}
			}
		}
	} else {
		logrus.Warnf("volume '%s' from '%s' is not volume.DetailedVolume", vol.Name(), vol.DriverName())
	}

	return true
}

// parsePattern returns the literal segment to match
func parsePattern(segment string) string {
	if len(segment) > 0 {
		if segment[0] == '!' || (len(segment) > 1 && segment[0] == '\\') {
			return segment[1:]
		}
	}
	return segment
}

// isNegation is true if the supplied segment contains a single ! at
// its beginning. To use a literal single ! prefix, the pattern must begin \!.
func isNegation(segment string) bool {
	if len(segment) > 0 && segment[0] == '!' {
		return true
	}
	return false
}

// elementOf checks if needle is present in set
func elementOf(needle string, set []string) bool {
	for _, el := range set {
		if needle == el {
			return true
		}
	}
	return false
}

// DetachMounts detaches mounts from a mount point plugin chain
func (c *Chain) DetachMounts(container string, mounts map[string]*MountPoint) error {
	var list []*MountPoint
	for _, mp := range mounts {
		list = append(list, mp)
	}
	return unwind(container, list)
}

// unwindAttachOnErr will clean up previous attachments if an error
// occurs during attachment
func (c *Chain) unwindAttachOnErr(pluginName, container string, mounts []*MountPoint, err error) (ret error) {
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
	var plugin *Plugin
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
			request := &DetachRequest{container}
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
func (c *Chain) SetPlugins(names []string) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.plugins, err = newPlugins(names); err != nil {
		return err
	}
	return nil
}

// DisableMountPointPlugin removes the mount point plugin from the chain
func (c *Chain) DisableMountPointPlugin(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// TODO: is it OK that this just removes it from further use?
	// There may still be mounts which are relying on it during tear
	// down
	var plugins []Plugin
	for _, plugin := range c.plugins {
		if plugin.Name() != name {
			plugins = append(plugins, plugin)
		}
	}
	c.plugins = plugins
}
