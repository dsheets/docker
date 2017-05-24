package mountpoint

import (
	mounttypes "github.com/docker/docker/api/types/mount"
)

const (
	// MountPointAPIProperties is the url for plugin properties queries
	MountPointAPIProperties = "MountPointPlugin.MountPointProperties"

	// MountPointAPIAttach is the url for mount point attachment interposition
	MountPointAPIAttach = "MountPointPlugin.MountPointAttach"

	// MountPointAPIDetach is the url for mount point detachment interposition
	MountPointAPIDetach = "MountPointPlugin.MountPointDetach"

	// MountPointAPIImplements is the name of the interface all mount point plugins implement
	MountPointAPIImplements = "mountpoint"
)

// PropertiesRequest holds a mount point plugin properties query
type PropertiesRequest struct {
}

// PropertiesResponse holds some static properties of the plugin
type PropertiesResponse struct {
	// Success indicates whether the properties query was successful
	Success bool

	// Types lists the types of mount points that this plugin interposes
	Types map[mounttypes.Type]bool

	// VolumePatterns returns a list of volume type patterns that this plugin interposes
	VolumePatterns []VolumePattern `json:",omitempty"`

	// Err stores a message in case there's an error
	Err string `json:",omitempty"`
}

// VolumePattern is a conjunction of predicates that must match a volume mount
type VolumePattern struct {
	// VolumePlugin is the volume plugin name that must be matched
	VolumePlugin string
	// OptionPattern is a set of predicates that a mount from volume plugin VolumePlugin must satisfy
	OptionPattern OptionPattern `json:",omitempty"`
}

// OptionPattern : all of the keys must be present and each value must
// match a comma-separated segment of the corresponding key; keys and
// values may begin with a '!' to invert the match or '\!' to match a
// key or value beginning with '!'.
type OptionPattern map[string][]string

// AttachRequest holds data required for mount point plugin attachment interposition
type AttachRequest struct {
	ID     string
	Mounts []*MountPoint
}

// AttachResponse represents mount point plugin response
type AttachResponse struct {
	// Success indicates whether the mount point was successful
	Success bool

	// Attachments contains information about the plugin's participation with the mount
	Attachments []Attachment `json:",omitempty"`

	// Err stores a message in case there's an error
	Err string `json:",omitempty"`
}

// Attachment describes how the plugin will interact with the mount
type Attachment struct {
	Attach        bool
	NewMountPoint string `json:",omitempty"`
}

// DetachRequest holds data required for mount point plugin detachment interposition
type DetachRequest struct {
	ID string
}

// DetachResponse represents mount point plugin response
type DetachResponse struct {
	// Success indicates whether detaching the mount point was successful
	Success bool

	// Recoverable indicates whether the failure (if any) is fatal to detach unwinding (false, default) or merely a container failure (true)
	Recoverable bool `json:",omitempty"`

	// Err stores a message in case there's an error
	Err string `json:",omitempty"`
}
