package volume

import (
	"fmt"

	"github.com/docker/docker/volume/mountpoint"
)

// AppliedMountPointMiddleware represents a mount point middleware's
// application to a specific mount point. It tracks which middleware
// was applied (both referentially and directly -- necessary for
// serialization/deserialization), how the middleware changed the
// mount point if any, and the order of mount point application.
type AppliedMountPointMiddleware struct {
	Name       string                          // Name is the name of the middleware (for later lookup)
	middleware *mountpoint.Middleware          // middleware stores the middleware object
	Attachment mountpoint.MountPointAttachment // Attachment contains whatever changes the middleware has made to the mount
	Clock      int                             // Clock is a positive integer used to ensure mount detachments occur in the correct order
}

// Middleware will retrieve the Middleware object or create a new one if none is available
func (p AppliedMountPointMiddleware) Middleware() (*mountpoint.Middleware, error) {
	if p.middleware == nil {
		pname := mountpoint.PluginNameOfMiddlewareName(p.Name)
		if pname == "" {
			return nil, fmt.Errorf("non-plugin middleware %s not found", p.Name)
		} else {
			plugin, err := mountpoint.NewMountPointPlugin(pname)
			if err != nil {
				return nil, err
			}
			middleware := mountpoint.Middleware(plugin)
			p.middleware = &middleware
		}
	}
	return p.middleware, nil
}

// EffectiveSource is the directory to use for a mount even after
// middleware may have changed the original source directory
func (m *MountPoint) EffectiveSource() string {
	for i := len(m.AppliedMiddleware) - 1; i >= 0; i-- {
		appliedMiddleware := m.AppliedMiddleware[i]
		if appliedMiddleware.Attachment.EffectiveSource != "" {
			return appliedMiddleware.Attachment.EffectiveSource
		}
	}
	return m.Source
}

// PushMiddleware pushes a new applied middleware onto the mount point's
// middleware stack
func (m *MountPoint) PushMiddleware(middleware mountpoint.Middleware, attachment mountpoint.MountPointAttachment, clock int) {
	appliedMiddleware := AppliedMountPointMiddleware{
		Name:       middleware.Name(),
		middleware: &middleware,
		Attachment: attachment,
		Clock:      clock,
	}
	m.AppliedMiddleware = append(m.AppliedMiddleware, appliedMiddleware)
}

// PopMiddleware removes and returns a middleware from the mount point's
// middleware stack
func (m *MountPoint) PopMiddleware() *AppliedMountPointMiddleware {
	stack := m.AppliedMiddleware
	if len(stack) > 0 {
		middleware := &stack[len(stack)-1]
		m.AppliedMiddleware = stack[0 : len(stack)-1]
		return middleware
	}
	return nil
}

// TopClock returns the Clock value from the middleware on the top of the
// mount point's middleware stack or 0 if the stack is empty
func (m *MountPoint) TopClock() int {
	stackSize := len(m.AppliedMiddleware)
	if stackSize > 0 {
		return m.AppliedMiddleware[stackSize-1].Clock
	}
	return 0
}
