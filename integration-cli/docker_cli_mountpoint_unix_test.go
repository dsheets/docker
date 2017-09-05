// +build !windows

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/docker/docker/integration-cli/checker"
	"github.com/docker/docker/integration-cli/daemon"
	"github.com/docker/docker/pkg/plugins"
	"github.com/docker/docker/volume/mountpoint"
	"github.com/go-check/check"
)

const (
	testMountPointPlugin = "mountpointplugin"
	mountFailMessage     = "mount source path contains 'secret'"
)

func init() {
	check.Suite(&DockerMountPointSuite{
		ds: &DockerSuite{},
	})
}

type DockerMountPointSuite struct {
	server [5]*httptest.Server
	ds     *DockerSuite
	d      *daemon.Daemon
	ctrl   [5]*mountPointController
	events []string
}

type mountPointController struct {
	propertiesRes mountpoint.PropertiesResponse // propertiesRes holds the plugin response to properties requests
	attachRes     mountpoint.AttachResponse     // attachRes holds the plugin response to attach requests
	detachRes     mountpoint.DetachResponse     // detachRes holds the plugin response to detach requests
	attachCnt     int                           // attachCnt counts the number of attach requests received
	attachMounts  [][]*mountpoint.MountPoint    // attachMounts is a stack of mount point sets requested for attachment
}

func (s *DockerMountPointSuite) SetUpTest(c *check.C) {
	s.d = daemon.New(c, dockerBinary, dockerdBinary, daemon.Config{
		Experimental: testEnv.ExperimentalDaemon(),
	})

	s.ctrl[0] = &mountPointController{
		attachRes: mountpoint.AttachResponse{
			Success: true,
		},
		detachRes: mountpoint.DetachResponse{
			Success: true,
		},
		attachCnt:    0,
		attachMounts: [][]*mountpoint.MountPoint{},
	}
	s.ctrl[1] = &mountPointController{}
	*s.ctrl[1] = *s.ctrl[0]
	s.ctrl[2] = &mountPointController{}
	*s.ctrl[2] = *s.ctrl[0]
	s.ctrl[3] = &mountPointController{}
	*s.ctrl[3] = *s.ctrl[0]
	s.ctrl[4] = &mountPointController{}
	*s.ctrl[4] = *s.ctrl[0]

	typeBind := mountpoint.TypeBind
	typeVolume := mountpoint.TypeVolume

	// matches -v /host:/container
	s.ctrl[0].propertiesRes = mountpoint.PropertiesResponse{
		Success: true,
		Patterns: []mountpoint.Pattern{
			{Type: &typeBind},
		},
	}
	// matches -v /host:/container AND all local volume mounts
	s.ctrl[1].propertiesRes = mountpoint.PropertiesResponse{
		Success: true,
		Patterns: []mountpoint.Pattern{
			{Type: &typeBind},
			{
				Type:   &typeVolume,
				Driver: []mountpoint.StringPattern{{Exactly: "local"}},
			},
		},
	}
	// matches local volume bind mounts (but not -v /container mounts)
	s.ctrl[2].propertiesRes = mountpoint.PropertiesResponse{
		Success: true,
		Patterns: []mountpoint.Pattern{
			{
				Type:   &typeVolume,
				Driver: []mountpoint.StringPattern{{Exactly: "local"}},
				Options: []mountpoint.StringMapPattern{{
					Exists: []mountpoint.StringMapKeyValuePattern{{
						Key:   mountpoint.StringPattern{Exactly: "o"},
						Value: mountpoint.StringPattern{Contains: "bind"},
					}},
				}},
			},
		},
	}
	// matches -v /container
	s.ctrl[3].propertiesRes = mountpoint.PropertiesResponse{
		Success: true,
		Patterns: []mountpoint.Pattern{
			{
				Type:   &typeVolume,
				Driver: []mountpoint.StringPattern{{Exactly: "local"}},
				Options: []mountpoint.StringMapPattern{{
					Not: true,
					Exists: []mountpoint.StringMapKeyValuePattern{
						{Key: mountpoint.StringPattern{Exactly: "o"}},
						{Key: mountpoint.StringPattern{Exactly: "device"}},
						{Key: mountpoint.StringPattern{Exactly: "type"}},
					},
				}},
			},
		},
	}
	// matches all bind mounts
	s.ctrl[4].propertiesRes = mountpoint.PropertiesResponse{
		Success: true,
		Patterns: []mountpoint.Pattern{
			{Type: &typeBind},
			{
				Type:   &typeVolume,
				Driver: []mountpoint.StringPattern{{Exactly: "local"}},
				Options: []mountpoint.StringMapPattern{{
					Exists: []mountpoint.StringMapKeyValuePattern{{
						Key:   mountpoint.StringPattern{Exactly: "o"},
						Value: mountpoint.StringPattern{Contains: "bind"},
					}},
				}},
			},
		},
	}

	s.events = []string{}
}

func (s *DockerMountPointSuite) TearDownTest(c *check.C) {
	if s.d != nil {
		s.d.Stop(c)
		s.ds.TearDownTest(c)
		s.ctrl[0] = nil
		s.ctrl[1] = nil
		s.ctrl[2] = nil
		s.ctrl[3] = nil
		s.ctrl[4] = nil

		//logs, err := s.d.ReadLogFile()
		//c.Assert(err, check.IsNil)
		//fmt.Print(string(logs))
	}
}

func (s *DockerMountPointSuite) SetUpSuite(c *check.C) {
	s.setupPlugin(c, 0)
	s.setupPlugin(c, 1)
	s.setupPlugin(c, 2)
	s.setupPlugin(c, 3)
	s.setupPlugin(c, 4)
}

func (s *DockerMountPointSuite) setupPlugin(c *check.C, i int) {
	mux := http.NewServeMux()
	s.server[i] = httptest.NewServer(mux)

	mux.HandleFunc("/Plugin.Activate", func(w http.ResponseWriter, r *http.Request) {
		b, err := json.Marshal(plugins.Manifest{Implements: []string{mountpoint.MountPointAPIImplements}})
		c.Assert(err, check.IsNil)
		w.Write(b)
	})

	mux.HandleFunc("/MountPointPlugin.MountPointProperties", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		c.Assert(err, check.IsNil)
		propertiesReq := mountpoint.PropertiesRequest{}
		err = json.Unmarshal(body, &propertiesReq)
		c.Assert(err, check.IsNil)

		s.events = append(s.events, fmt.Sprintf("%d:properties", i))

		propertiesRes := s.ctrl[i].propertiesRes
		if !propertiesRes.Success {
			w.WriteHeader(http.StatusInternalServerError)
		}
		b, err := json.Marshal(propertiesRes)
		c.Assert(err, check.IsNil)
		w.Write(b)
	})

	mux.HandleFunc("/MountPointPlugin.MountPointAttach", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		c.Assert(err, check.IsNil)
		attachReq := mountpoint.AttachRequest{}
		err = json.Unmarshal(body, &attachReq)
		c.Assert(err, check.IsNil)

		s.ctrl[i].attachCnt++
		s.ctrl[i].attachMounts = append(s.ctrl[i].attachMounts, attachReq.Mounts)
		s.events = append(s.events, fmt.Sprintf("%d:attach", i))

		attachRes := s.ctrl[i].attachRes
		if !attachRes.Success {
			w.WriteHeader(http.StatusInternalServerError)
		}
		b, err := json.Marshal(attachRes)
		c.Assert(err, check.IsNil)
		w.Write(b)
	})

	mux.HandleFunc("/MountPointPlugin.MountPointDetach", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		c.Assert(err, check.IsNil)
		detachReq := mountpoint.DetachRequest{}
		err = json.Unmarshal(body, &detachReq)
		c.Assert(err, check.IsNil)

		s.events = append(s.events, fmt.Sprintf("%d:detach", i))

		detachRes := s.ctrl[i].detachRes
		if !detachRes.Success && !detachRes.Recoverable {
			w.WriteHeader(http.StatusInternalServerError)
		}
		b, err := json.Marshal(detachRes)
		c.Assert(err, check.IsNil)
		w.Write(b)
	})

	err := os.MkdirAll("/etc/docker/plugins", 0755)
	c.Assert(err, checker.IsNil)

	fileName := fmt.Sprintf("/etc/docker/plugins/%s%d.spec", testMountPointPlugin, i)
	err = ioutil.WriteFile(fileName, []byte(s.server[i].URL), 0644)
	c.Assert(err, checker.IsNil)
}

func (s *DockerMountPointSuite) TearDownSuite(c *check.C) {
	for i := 0; i < 5; i++ {
		if s.server[i] == nil {
			continue
		}

		s.server[i].Close()
	}

	err := os.RemoveAll("/etc/docker/plugins")
	c.Assert(err, checker.IsNil)
}

func (s *DockerMountPointSuite) TestMountPointPluginNoMounts(c *check.C) {
	s.d.Start(c, fmt.Sprintf("--mount-point-plugin=%s0", testMountPointPlugin))
	s.d.LoadBusybox(c)

	// Ensure command successful
	out, err := s.d.Cmd("run", "-d", "busybox", "top")
	c.Assert(err, check.IsNil)

	id := strings.TrimSpace(out)

	out, err = s.d.Cmd("ps")
	c.Assert(err, check.IsNil)
	c.Assert(assertContainerList(out, []string{id}), check.Equals, true)
	c.Assert(s.ctrl[0].attachCnt, check.Equals, 0)
}

func (s *DockerMountPointSuite) TestMountPointPluginError(c *check.C) {
	s.d.Start(c, fmt.Sprintf("--mount-point-plugin=%s0", testMountPointPlugin))
	s.d.LoadBusybox(c)

	s.ctrl[0].attachRes = mountpoint.AttachResponse{
		Success: false,
		Err:     mountFailMessage,
	}

	res, err := s.d.Cmd("run", "-d", "-v", "/secret:/host", "busybox", "top")
	c.Assert(err, check.NotNil, check.Commentf(res))

	c.Assert(res, checker.HasSuffix, fmt.Sprintf("Error response from daemon: middleware plugin:%s0 failed with error: %s: %s.\n", testMountPointPlugin, mountpoint.MountPointAPIAttach, mountFailMessage))
	c.Assert(s.ctrl[0].attachCnt, check.Equals, 1)
}

func (s *DockerMountPointSuite) TestMountPointPluginFilter(c *check.C) {
	s.d.Start(c, fmt.Sprintf("--mount-point-plugin=%s0", testMountPointPlugin))
	s.d.LoadBusybox(c)

	// Ensure command successful
	out, err := s.d.Cmd("run", "-d", "-v", "/host", "busybox", "top")
	c.Assert(err, check.IsNil)

	id := strings.TrimSpace(out)

	out, err = s.d.Cmd("ps")
	c.Assert(err, check.IsNil)
	c.Assert(assertContainerList(out, []string{id}), check.Equals, true)
	c.Assert(s.ctrl[0].attachCnt, check.Equals, 0)
}

func (s *DockerMountPointSuite) TestMountPointPluginEnsureNoDuplicatePluginRegistration(c *check.C) {
	s.d.Start(c, fmt.Sprintf("--mount-point-plugin=%s0", testMountPointPlugin), fmt.Sprintf("--mount-point-plugin=%s0", testMountPointPlugin))
	s.d.LoadBusybox(c)

	out, err := s.d.Cmd("run", "-d", "-v", "/:/host", "busybox", "top")
	c.Assert(err, check.IsNil, check.Commentf(out))

	id := strings.TrimSpace(out)

	out, err = s.d.Cmd("ps")
	c.Assert(err, check.IsNil)
	c.Assert(assertContainerList(out, []string{id}), check.Equals, true)

	// assert plugin is only called once
	c.Assert(s.ctrl[0].attachCnt, check.Equals, 1)
}

func (s *DockerMountPointSuite) TestMountPointPluginAttachOrderBind(c *check.C) {
	s.d.Start(c,
		fmt.Sprintf("--mount-point-plugin=%s0", testMountPointPlugin),
		fmt.Sprintf("--mount-point-plugin=%s1", testMountPointPlugin),
		fmt.Sprintf("--mount-point-plugin=%s0", testMountPointPlugin))
	s.d.LoadBusybox(c)

	out, err := s.d.Cmd("run", "-d", "-v", "/:/host", "busybox", "top")
	c.Assert(err, check.IsNil, check.Commentf(out))

	id := strings.TrimSpace(out)

	out, err = s.d.Cmd("ps")
	c.Assert(err, check.IsNil)
	c.Assert(assertContainerList(out, []string{id}), check.Equals, true)

	c.Assert(s.events, checker.DeepEquals, []string{"0:properties", "1:properties", "0:attach", "1:attach"})
}

func (s *DockerMountPointSuite) TestMountPointPluginVolumeFilter(c *check.C) {
	s.d.Start(c,
		fmt.Sprintf("--mount-point-plugin=%s0", testMountPointPlugin),
		fmt.Sprintf("--mount-point-plugin=%s1", testMountPointPlugin),
		fmt.Sprintf("--mount-point-plugin=%s2", testMountPointPlugin),
		fmt.Sprintf("--mount-point-plugin=%s3", testMountPointPlugin))
	s.d.LoadBusybox(c)

	out, err := s.d.Cmd("run", "-d", "-v", "/anon", "busybox", "top")
	c.Assert(err, check.IsNil, check.Commentf(out))

	id := strings.TrimSpace(out)

	out, err = s.d.Cmd("ps")
	c.Assert(err, check.IsNil)
	c.Assert(assertContainerList(out, []string{id}), check.Equals, true)

	c.Assert(s.events, checker.DeepEquals, []string{"0:properties", "1:properties", "2:properties", "3:properties", "1:attach", "3:attach"})
}

func (s *DockerMountPointSuite) TestMountPointPluginLocalFilter(c *check.C) {
	s.d.Start(c,
		fmt.Sprintf("--mount-point-plugin=%s0", testMountPointPlugin),
		fmt.Sprintf("--mount-point-plugin=%s1", testMountPointPlugin),
		fmt.Sprintf("--mount-point-plugin=%s2", testMountPointPlugin))
	s.d.LoadBusybox(c)

	out, err := s.d.Cmd("volume", "create", "--opt", "type=tmpfs", "--opt", "device=tmpfs")
	c.Assert(err, check.IsNil, check.Commentf(out))

	volID := strings.TrimSpace(out)

	out, err = s.d.Cmd("run", "-d", "-v", volID+":/tmpfs", "busybox", "top")
	c.Assert(err, check.IsNil, check.Commentf(out))

	id := strings.TrimSpace(out)

	out, err = s.d.Cmd("ps")
	c.Assert(err, check.IsNil)
	c.Assert(assertContainerList(out, []string{id}), check.Equals, true)

	c.Assert(s.events, checker.DeepEquals, []string{"0:properties", "1:properties", "2:properties", "1:attach"})
}

func (s *DockerMountPointSuite) TestMountPointPluginLocalBindFilter(c *check.C) {
	s.d.Start(c,
		fmt.Sprintf("--mount-point-plugin=%s0", testMountPointPlugin),
		fmt.Sprintf("--mount-point-plugin=%s1", testMountPointPlugin),
		fmt.Sprintf("--mount-point-plugin=%s2", testMountPointPlugin))
	s.d.LoadBusybox(c)

	out, err := s.d.Cmd("volume", "create", "--opt", "device=/etc", "--opt", "o=ro,bind")
	c.Assert(err, check.IsNil, check.Commentf(out))

	volID := strings.TrimSpace(out)

	out, err = s.d.Cmd("run", "-d", "-v", volID+":/tmpfs", "busybox", "top")
	c.Assert(err, check.IsNil, check.Commentf(out))

	id := strings.TrimSpace(out)

	out, err = s.d.Cmd("ps")
	c.Assert(err, check.IsNil)
	c.Assert(assertContainerList(out, []string{id}), check.Equals, true)

	c.Assert(s.events, checker.DeepEquals, []string{"0:properties", "1:properties", "2:properties", "1:attach", "2:attach"})
}

func (s *DockerMountPointSuite) TestMountPointPluginChangeDirectory(c *check.C) {
	s.d.Start(c,
		fmt.Sprintf("--mount-point-plugin=%s1", testMountPointPlugin),
		fmt.Sprintf("--mount-point-plugin=%s3", testMountPointPlugin))
	s.d.LoadBusybox(c)

	newdir := "/var/run/" + testMountPointPlugin + "1/newdir"
	err := os.MkdirAll(newdir, 0700)
	c.Assert(err, check.IsNil)
	s.ctrl[1].attachRes = mountpoint.AttachResponse{
		Success: true,
		Attachments: []mountpoint.Attachment{
			{
				Attach: true,
				Changes: mountpoint.Changes{
					EffectiveSource: newdir,
				},
			},
		},
	}

	out, err := s.d.Cmd("run", "-d", "-v", "/anon", "busybox", "top")
	c.Assert(err, check.IsNil, check.Commentf(out))

	id := strings.TrimSpace(out)

	out, err = s.d.Cmd("ps")
	c.Assert(err, check.IsNil)
	c.Assert(assertContainerList(out, []string{id}), check.Equals, true)

	c.Assert(s.events, checker.DeepEquals, []string{"1:properties", "3:properties", "1:attach", "3:attach"})

	c.Assert(len(s.ctrl[3].attachMounts), check.Equals, 1)
	c.Assert(len(s.ctrl[3].attachMounts[0]), check.Equals, 1)
	c.Assert(s.ctrl[3].attachMounts[0][0].EffectiveSource, check.Equals, newdir)
}

func (s *DockerMountPointSuite) TestMountPointPluginFailureUnwind(c *check.C) {
	s.d.Start(c,
		fmt.Sprintf("--mount-point-plugin=%s1", testMountPointPlugin),
		fmt.Sprintf("--mount-point-plugin=%s2", testMountPointPlugin),
		fmt.Sprintf("--mount-point-plugin=%s4", testMountPointPlugin))
	s.d.LoadBusybox(c)

	s.ctrl[1].attachRes = mountpoint.AttachResponse{
		Success: true,
		Attachments: []mountpoint.Attachment{
			{
				Attach: true,
			},
		},
	}
	s.ctrl[2].attachRes = mountpoint.AttachResponse{
		Success: true,
		Attachments: []mountpoint.Attachment{
			{
				Attach: true,
			},
		},
	}
	s.ctrl[4].attachRes = mountpoint.AttachResponse{
		Success: false,
		Err:     mountFailMessage,
	}

	out, err := s.d.Cmd("volume", "create", "--opt", "device=/etc", "--opt", "o=ro,bind")
	c.Assert(err, check.IsNil, check.Commentf(out))

	volID := strings.TrimSpace(out)

	out, err = s.d.Cmd("run", "-d", "-v", volID+":/tmpfs", "busybox", "top")
	c.Assert(err, check.NotNil, check.Commentf(out))

	c.Assert(out, checker.HasSuffix, fmt.Sprintf("Error response from daemon: middleware plugin:%s4 failed with error: %s: %s.\n", testMountPointPlugin, mountpoint.MountPointAPIAttach, mountFailMessage))

	c.Assert(s.events, checker.DeepEquals, []string{"1:properties", "2:properties", "4:properties", "1:attach", "2:attach", "4:attach", "2:detach", "1:detach"})

	s.ctrl[2].attachRes.Attachments[0].Attach = false
	s.events = []string{}

	out, err = s.d.Cmd("run", "-d", "-v", volID+":/tmpfs", "busybox", "top")
	c.Assert(err, check.NotNil, check.Commentf(out))

	c.Assert(out, checker.HasSuffix, fmt.Sprintf("Error response from daemon: middleware plugin:%s4 failed with error: %s: %s.\n", testMountPointPlugin, mountpoint.MountPointAPIAttach, mountFailMessage))

	c.Assert(s.events, checker.DeepEquals, []string{"1:attach", "2:attach", "4:attach", "1:detach"})
}

func (s *DockerMountPointSuite) TestMountPointPluginDetachExit(c *check.C) {
	s.d.Start(c,
		fmt.Sprintf("--mount-point-plugin=%s1", testMountPointPlugin),
		fmt.Sprintf("--mount-point-plugin=%s2", testMountPointPlugin),
		fmt.Sprintf("--mount-point-plugin=%s4", testMountPointPlugin))
	s.d.LoadBusybox(c)

	s.ctrl[1].attachRes = mountpoint.AttachResponse{
		Success: true,
		Attachments: []mountpoint.Attachment{
			{
				Attach: true,
			},
		},
	}
	s.ctrl[4].attachRes = s.ctrl[1].attachRes

	out, err := s.d.Cmd("volume", "create", "--opt", "device=/etc", "--opt", "o=ro,bind")
	c.Assert(err, check.IsNil, check.Commentf(out))

	volID := strings.TrimSpace(out)

	out, err = s.d.Cmd("run", "-d", "-v", volID+":/tmpfs", "busybox", "true")
	c.Assert(err, check.IsNil)

	id := strings.TrimSpace(out)

	_, err = s.d.Cmd("wait", id)
	c.Assert(err, check.IsNil)

	c.Assert(s.events, checker.DeepEquals, []string{"1:properties", "2:properties", "4:properties", "1:attach", "2:attach", "4:attach", "4:detach", "1:detach"})
}

func (s *DockerMountPointSuite) TestMountPointPluginMultipleMounts(c *check.C) {
	s.d.Start(c,
		fmt.Sprintf("--mount-point-plugin=%s0", testMountPointPlugin),
		fmt.Sprintf("--mount-point-plugin=%s1", testMountPointPlugin),
		fmt.Sprintf("--mount-point-plugin=%s2", testMountPointPlugin))
	s.d.LoadBusybox(c)

	s.ctrl[0].attachRes = mountpoint.AttachResponse{
		Success: true,
		Attachments: []mountpoint.Attachment{
			{
				Attach: true,
				Changes: mountpoint.Changes{
					EffectiveSource: "/usr",
				},
			},
			{ // tests overlong attach lists
				Attach: true,
			},
		},
	}
	s.ctrl[1].attachRes = mountpoint.AttachResponse{
		Success: true,
		Attachments: []mountpoint.Attachment{
			{
				Attach: true,
				Changes: mountpoint.Changes{
					EffectiveSource: "/etc",
				},
			},
			{
				Attach: true,
			},
		},
	}
	s.ctrl[2].attachRes = mountpoint.AttachResponse{
		Success: true,
		Attachments: []mountpoint.Attachment{
			{
				Attach: true,
			},
		},
	}

	out, err := s.d.Cmd("volume", "create", "--opt", "device=/etc", "--opt", "o=ro,bind")
	c.Assert(err, check.IsNil, check.Commentf(out))

	volID := strings.TrimSpace(out)

	out, err = s.d.Cmd("run", "-d",
		"-v", volID+":/host_etc",
		"-v", "/:/host",
		"busybox", "true")
	c.Assert(err, check.IsNil)

	id := strings.TrimSpace(out)

	_, err = s.d.Cmd("wait", id)
	c.Assert(err, check.IsNil)

	c.Assert(s.events, checker.DeepEquals, []string{"0:properties", "1:properties", "2:properties", "0:attach", "1:attach", "2:attach", "2:detach", "1:detach", "0:detach"})

	c.Assert(len(s.ctrl[0].attachMounts), check.Equals, 1)
	c.Assert(len(s.ctrl[0].attachMounts[0]), check.Equals, 1)
	c.Assert(s.ctrl[0].attachMounts[0][0].EffectiveSource, check.Equals, "/")

	c.Assert(len(s.ctrl[1].attachMounts), check.Equals, 1)
	c.Assert(len(s.ctrl[1].attachMounts[0]), check.Equals, 2)
	c.Assert(s.ctrl[1].attachMounts[0][1].EffectiveSource, check.Equals, "/usr")

	c.Assert(len(s.ctrl[2].attachMounts), check.Equals, 1)
	c.Assert(len(s.ctrl[2].attachMounts[0]), check.Equals, 1)
	c.Assert(s.ctrl[2].attachMounts[0][0].EffectiveSource, check.Equals, "/etc")
}

func (s *DockerMountPointSuite) TestMountPointPluginDetachCleanFailure(c *check.C) {
	s.d.Start(c,
		fmt.Sprintf("--mount-point-plugin=%s0", testMountPointPlugin),
		fmt.Sprintf("--mount-point-plugin=%s1", testMountPointPlugin))
	s.d.LoadBusybox(c)

	s.ctrl[0].attachRes = mountpoint.AttachResponse{
		Success: true,
		Attachments: []mountpoint.Attachment{
			{
				Attach: true,
			},
		},
	}
	s.ctrl[1].attachRes = s.ctrl[0].attachRes

	s.ctrl[1].detachRes = mountpoint.DetachResponse{
		Success:     false,
		Recoverable: true,
		Err:         "kaboom",
	}

	out, err := s.d.Cmd("run", "-d", "-v", "/:/host", "busybox", "true")
	c.Assert(err, check.IsNil)

	id := strings.TrimSpace(out)

	out, err = s.d.Cmd("wait", id)
	c.Assert(err, check.IsNil)

	exitCode := strings.TrimSpace(out)
	c.Assert(exitCode, check.Equals, "129")

	c.Assert(s.events, checker.DeepEquals, []string{"0:properties", "1:properties", "0:attach", "1:attach", "1:detach", "0:detach"})
}

func (s *DockerMountPointSuite) TestMountPointPluginStopStart(c *check.C) {
	s.d.Start(c,
		fmt.Sprintf("--mount-point-plugin=%s0", testMountPointPlugin),
		fmt.Sprintf("--mount-point-plugin=%s1", testMountPointPlugin))
	s.d.LoadBusybox(c)

	s.ctrl[0].attachRes = mountpoint.AttachResponse{
		Success: true,
		Attachments: []mountpoint.Attachment{
			{
				Attach: true,
			},
		},
	}
	s.ctrl[1].attachRes = s.ctrl[0].attachRes

	out, err := s.d.Cmd("run", "-d", "-v", "/:/host", "busybox", "top")
	c.Assert(err, check.IsNil)

	id := strings.TrimSpace(out)

	_, err = s.d.Cmd("stop", id)
	c.Assert(err, check.IsNil)

	_, err = s.d.Cmd("start", id)
	c.Assert(err, check.IsNil)

	c.Assert(s.events, checker.DeepEquals, []string{"0:properties", "1:properties", "0:attach", "1:attach", "1:detach", "0:detach", "0:attach", "1:attach"})
}

func (s *DockerMountPointSuite) TestMountPointPluginKill(c *check.C) {
	s.d.Start(c,
		fmt.Sprintf("--mount-point-plugin=%s0", testMountPointPlugin),
		fmt.Sprintf("--mount-point-plugin=%s1", testMountPointPlugin))
	s.d.LoadBusybox(c)

	s.ctrl[0].attachRes = mountpoint.AttachResponse{
		Success: true,
		Attachments: []mountpoint.Attachment{
			{
				Attach: true,
			},
		},
	}
	s.ctrl[1].attachRes = s.ctrl[0].attachRes

	out, err := s.d.Cmd("run", "-d", "-v", "/:/host", "busybox", "top")
	c.Assert(err, check.IsNil)

	id := strings.TrimSpace(out)

	_, err = s.d.Cmd("kill", id)
	c.Assert(err, check.IsNil)

	c.Assert(s.events, checker.DeepEquals, []string{"0:properties", "1:properties", "0:attach", "1:attach", "1:detach", "0:detach"})
}

func (s *DockerMountPointSuite) TestMountPointPluginOOM(c *check.C) {
	testRequires(c, DaemonIsLinux, memoryLimitSupport, swapMemorySupport)

	s.d.Start(c,
		fmt.Sprintf("--mount-point-plugin=%s0", testMountPointPlugin),
		fmt.Sprintf("--mount-point-plugin=%s1", testMountPointPlugin))
	s.d.LoadBusybox(c)

	s.ctrl[0].attachRes = mountpoint.AttachResponse{
		Success: true,
		Attachments: []mountpoint.Attachment{
			{
				Attach: true,
			},
		},
	}
	s.ctrl[1].attachRes = s.ctrl[0].attachRes

	out, err := s.d.Cmd("run", "-d", "-v", "/:/host", "--memory", "32MB", "busybox", "sh", "-c", "x=a; while true; do x=$x$x$x$x; done")
	c.Assert(err, check.IsNil)

	id := strings.TrimSpace(out)

	out, err = s.d.Cmd("wait", id)

	exitCode := strings.TrimSpace(out)

	c.Assert(exitCode, checker.Equals, "137", check.Commentf("OOM exit should be 137"))

	c.Assert(s.events, checker.DeepEquals, []string{"0:properties", "1:properties", "0:attach", "1:attach", "1:detach", "0:detach"})
}

func (s *DockerMountPointSuite) TestMountPointPluginDaemonRestart(c *check.C) {
	testRequires(c, DaemonIsLinux)

	s.d.Start(c, "--live-restore",
		fmt.Sprintf("--mount-point-plugin=%s0", testMountPointPlugin),
		fmt.Sprintf("--mount-point-plugin=%s1", testMountPointPlugin))
	s.d.LoadBusybox(c)

	s.ctrl[0].attachRes = mountpoint.AttachResponse{
		Success: true,
		Attachments: []mountpoint.Attachment{
			{
				Attach: true,
			},
		},
	}
	s.ctrl[1].attachRes = s.ctrl[0].attachRes

	out, err := s.d.Cmd("run", "-d", "-v", "/:/host", "busybox", "top")
	c.Assert(err, check.IsNil, check.Commentf("output: %s", out))
	id := strings.TrimSpace(out)

	// restart without attached plugins
	s.d.Restart(c, "--live-restore")

	out, err = s.d.Cmd("stop", id)
	c.Assert(err, check.IsNil, check.Commentf("output: %s", out))

	// uninitialized plugins are initialized before detach
	c.Assert(s.events, checker.DeepEquals, []string{"0:properties", "1:properties", "0:attach", "1:attach", "1:properties", "1:detach", "0:properties", "0:detach"})

	s.events = []string{}
	out, err = s.d.Cmd("run", "-d", "-v", "/:/host", "busybox", "top")
	c.Assert(err, check.IsNil, check.Commentf("output: %s", out))
	id = strings.TrimSpace(out)
	defer s.d.Cmd("stop", id)

	// no new plugin events have occurred without explicit plugin loading
	c.Assert(s.events, checker.DeepEquals, []string{})
}
