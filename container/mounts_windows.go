package container

import "github.com/docker/docker/volume"

// Mount contains information for a mount operation.
type Mount struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Writable    bool   `json:"writable"`
}

func MountOfMountPoint(m *volume.MountPoint) Mount {
	return Mount{
		Source:      m.Source,
		Destination: m.Destination,
		Writable:    m.RW,
	}
}
