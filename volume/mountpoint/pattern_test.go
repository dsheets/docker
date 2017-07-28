package mountpoint

import (
	"testing"

	"github.com/docker/docker/api/types/mount"
	"github.com/stretchr/testify/require"
)

func testStringPatternInverse(pattern StringPattern, f func(pattern StringPattern, tru, fals bool)) {
	f(pattern, true, false)
	pattern.Not = true
	f(pattern, false, true)
}

func TestStringPattern(t *testing.T) {
	pattern := StringPattern{}
	require.Equal(t, true, stringPatternMatches(pattern, ""))
	require.Equal(t, true, stringPatternMatches(pattern, "asdf"))

	pattern = StringPattern{Not: true}
	require.Equal(t, true, stringPatternMatches(pattern, ""))
	require.Equal(t, true, stringPatternMatches(pattern, "asdf"))
}

func TestStringPatternEmpty(t *testing.T) {
	testStringPatternInverse(StringPattern{Empty: true},
		func(pattern StringPattern, tru, fals bool) {
			require.Equal(t, tru, stringPatternMatches(pattern, ""))
			require.Equal(t, fals, stringPatternMatches(pattern, "asdf"))
		})
}

func TestStringPatternPrefix(t *testing.T) {
	testStringPatternInverse(StringPattern{Prefix: "as"},
		func(pattern StringPattern, tru, fals bool) {
			require.Equal(t, fals, stringPatternMatches(pattern, ""))
			require.Equal(t, fals, stringPatternMatches(pattern, "adsf"))
			require.Equal(t, tru, stringPatternMatches(pattern, "asdf"))
		})
}

func TestStringPatternPathPrefix(t *testing.T) {
	testStringPatternInverse(StringPattern{PathPrefix: "///a/./b/c/../foo"},
		func(pattern StringPattern, tru, fals bool) {
			require.Equal(t, fals, stringPatternMatches(pattern, "/a/b/fo"))
			require.Equal(t, tru, stringPatternMatches(pattern, "/a/b/foo"))
			require.Equal(t, tru, stringPatternMatches(pattern, "/a/b/foo/"))
			require.Equal(t, fals, stringPatternMatches(pattern, "/a/b/foobar"))
			require.Equal(t, tru, stringPatternMatches(pattern, "/a/b/foo/bar"))
			require.Equal(t, tru, stringPatternMatches(pattern, "/a//b/c/d/../../foo/./bar"))
		})
}

func TestStringPatternSuffix(t *testing.T) {
	testStringPatternInverse(StringPattern{Suffix: ".dat"},
		func(pattern StringPattern, tru, fals bool) {
			require.Equal(t, fals, stringPatternMatches(pattern, ""))
			require.Equal(t, tru, stringPatternMatches(pattern, ".dat"))
			require.Equal(t, fals, stringPatternMatches(pattern, "foo.dab"))
			require.Equal(t, tru, stringPatternMatches(pattern, "/xyz/foo.dat"))
		})
}

func TestStringPatternExactly(t *testing.T) {
	testStringPatternInverse(StringPattern{Exactly: "xyz"},
		func(pattern StringPattern, tru, fals bool) {
			require.Equal(t, fals, stringPatternMatches(pattern, ""))
			require.Equal(t, fals, stringPatternMatches(pattern, "xy"))
			require.Equal(t, fals, stringPatternMatches(pattern, "xyyz"))
			require.Equal(t, fals, stringPatternMatches(pattern, "wxyz"))
			require.Equal(t, fals, stringPatternMatches(pattern, "xyz0"))
			require.Equal(t, tru, stringPatternMatches(pattern, "xyz"))
		})
}

func TestStringPatternContains(t *testing.T) {
	testStringPatternInverse(StringPattern{Contains: "xyz"},
		func(pattern StringPattern, tru, fals bool) {
			require.Equal(t, fals, stringPatternMatches(pattern, ""))
			require.Equal(t, fals, stringPatternMatches(pattern, "xy"))
			require.Equal(t, fals, stringPatternMatches(pattern, "xyyz"))
			require.Equal(t, tru, stringPatternMatches(pattern, "wxyz"))
			require.Equal(t, tru, stringPatternMatches(pattern, "xyz0"))
			require.Equal(t, tru, stringPatternMatches(pattern, "xyz"))
		})
}

func testStringMapPatternInverse(pattern StringMapPattern, f func(pattern StringMapPattern, tru, fals bool)) {
	f(pattern, true, false)
	pattern.Not = true
	f(pattern, false, true)
}

func TestStringMapPatternExists(t *testing.T) {
	testStringMapPatternInverse(StringMapPattern{
		Exists: map[StringPattern]*StringPattern{
			{Exactly: "key"}: nil,
		},
	}, func(pattern StringMapPattern, tru, fals bool) {
		stringMap := map[string]string{}
		require.Equal(t, fals, stringMapPatternMatches(pattern, stringMap))
		stringMap = map[string]string{"foo": ""}
		require.Equal(t, fals, stringMapPatternMatches(pattern, stringMap))
		stringMap = map[string]string{
			"foo": "",
			"key": "",
		}
		require.Equal(t, tru, stringMapPatternMatches(pattern, stringMap))
		stringMap = map[string]string{
			"foo": "",
			"key": "value",
		}
		require.Equal(t, tru, stringMapPatternMatches(pattern, stringMap))
	})

	testStringMapPatternInverse(StringMapPattern{
		Exists: map[StringPattern]*StringPattern{
			{Exactly: "key"}: {Exactly: "value"},
		},
	}, func(pattern StringMapPattern, tru, fals bool) {
		stringMap := map[string]string{
			"foo": "",
			"key": "",
		}
		require.Equal(t, fals, stringMapPatternMatches(pattern, stringMap))
		stringMap = map[string]string{
			"foo": "",
			"key": "value",
		}
		require.Equal(t, tru, stringMapPatternMatches(pattern, stringMap))
	})

	testStringMapPatternInverse(StringMapPattern{
		Exists: map[StringPattern]*StringPattern{
			{Exactly: "key"}: nil,
			{Exactly: "k2"}:  nil,
		},
	}, func(pattern StringMapPattern, tru, fals bool) {
		stringMap := map[string]string{}
		require.Equal(t, fals, stringMapPatternMatches(pattern, stringMap))
		stringMap = map[string]string{
			"key": "",
		}
		require.Equal(t, false, stringMapPatternMatches(pattern, stringMap))
		stringMap = map[string]string{
			"k2": "",
		}
		require.Equal(t, false, stringMapPatternMatches(pattern, stringMap))
		stringMap = map[string]string{
			"k2":  "",
			"key": "",
		}
		require.Equal(t, tru, stringMapPatternMatches(pattern, stringMap))
		stringMap = map[string]string{
			"k2":  "",
			"k3":  "",
			"key": "",
		}
		require.Equal(t, tru, stringMapPatternMatches(pattern, stringMap))
	})

	testStringMapPatternInverse(StringMapPattern{
		Exists: map[StringPattern]*StringPattern{
			{}:               {Prefix: "abc"},
			{Exactly: "key"}: {Suffix: "bcd"},
		},
	}, func(pattern StringMapPattern, tru, fals bool) {
		stringMap := map[string]string{}
		require.Equal(t, fals, stringMapPatternMatches(pattern, stringMap))
		stringMap = map[string]string{
			"key": "",
		}
		require.Equal(t, fals, stringMapPatternMatches(pattern, stringMap))
		stringMap = map[string]string{
			"key": "bcd",
		}
		require.Equal(t, false, stringMapPatternMatches(pattern, stringMap))
		stringMap = map[string]string{
			"key": "bcd",
			"k2":  "abc",
		}
		require.Equal(t, tru, stringMapPatternMatches(pattern, stringMap))
		stringMap = map[string]string{
			"key": "abcd",
		}
		require.Equal(t, tru, stringMapPatternMatches(pattern, stringMap))
	})
}

func TestStringMapPatternAll(t *testing.T) {
	testStringMapPatternInverse(StringMapPattern{
		All: map[StringPattern]*StringPattern{
			{Prefix: "pre"}: nil,
		},
	}, func(pattern StringMapPattern, tru, fals bool) {
		stringMap := map[string]string{}
		require.Equal(t, tru, stringMapPatternMatches(pattern, stringMap))
		stringMap = map[string]string{
			"foo": "",
		}
		require.Equal(t, fals, stringMapPatternMatches(pattern, stringMap))
		stringMap = map[string]string{
			"prefix": "",
		}
		require.Equal(t, tru, stringMapPatternMatches(pattern, stringMap))
		stringMap = map[string]string{
			"foo":    "",
			"prefix": "",
		}
		require.Equal(t, fals, stringMapPatternMatches(pattern, stringMap))
		stringMap = map[string]string{
			"precursor": "",
			"prefix":    "",
		}
		require.Equal(t, tru, stringMapPatternMatches(pattern, stringMap))
	})

	testStringMapPatternInverse(StringMapPattern{
		All: map[StringPattern]*StringPattern{
			{Prefix: "pre"}: nil,
			{Suffix: "x"}:   nil,
		},
	}, func(pattern StringMapPattern, tru, fals bool) {
		stringMap := map[string]string{}
		require.Equal(t, tru, stringMapPatternMatches(pattern, stringMap))
		stringMap = map[string]string{
			"quux": "",
		}
		require.Equal(t, false, stringMapPatternMatches(pattern, stringMap))
		stringMap = map[string]string{
			"prefix": "",
		}
		require.Equal(t, tru, stringMapPatternMatches(pattern, stringMap))
		stringMap = map[string]string{
			"quux":   "",
			"prefix": "",
		}
		require.Equal(t, false, stringMapPatternMatches(pattern, stringMap))
		stringMap = map[string]string{
			"prenex": "",
			"prefix": "",
		}
		require.Equal(t, tru, stringMapPatternMatches(pattern, stringMap))
	})

	testStringMapPatternInverse(StringMapPattern{
		All: map[StringPattern]*StringPattern{
			{Prefix: "key"}: {Exactly: "value"},
		},
	}, func(pattern StringMapPattern, tru, fals bool) {
		stringMap := map[string]string{}
		require.Equal(t, tru, stringMapPatternMatches(pattern, stringMap))
		stringMap = map[string]string{
			"key": "",
		}
		require.Equal(t, fals, stringMapPatternMatches(pattern, stringMap))
		stringMap = map[string]string{
			"key": "value",
		}
		require.Equal(t, tru, stringMapPatternMatches(pattern, stringMap))
		stringMap = map[string]string{
			"key0": "value",
			"key1": "",
		}
		require.Equal(t, fals, stringMapPatternMatches(pattern, stringMap))
		stringMap = map[string]string{
			"key0": "value",
			"key1": "value",
		}
		require.Equal(t, tru, stringMapPatternMatches(pattern, stringMap))
		stringMap = map[string]string{
			"key0": "value",
			"key1": "value",
			"k2":   "xyz",
		}
		require.Equal(t, tru, stringMapPatternMatches(pattern, stringMap))
	})

	testStringMapPatternInverse(StringMapPattern{
		All: map[StringPattern]*StringPattern{
			{Prefix: "key"}: {Prefix: "v"},
			{Suffix: "_"}:   {Suffix: "e"},
		},
	}, func(pattern StringMapPattern, tru, fals bool) {
		stringMap := map[string]string{}
		require.Equal(t, tru, stringMapPatternMatches(pattern, stringMap))
		stringMap = map[string]string{
			"key_": "",
		}
		require.Equal(t, fals, stringMapPatternMatches(pattern, stringMap))
		stringMap = map[string]string{
			"key_": "value",
		}
		require.Equal(t, tru, stringMapPatternMatches(pattern, stringMap))
		stringMap = map[string]string{
			"key0_": "value",
			"key1":  "",
		}
		require.Equal(t, false, stringMapPatternMatches(pattern, stringMap))
		stringMap = map[string]string{
			"key0_": "value",
			"key1_": "value",
		}
		require.Equal(t, tru, stringMapPatternMatches(pattern, stringMap))
		stringMap = map[string]string{
			"key0_": "value",
			"key1":  "val",
			"d2_":   "abcde",
		}
		require.Equal(t, tru, stringMapPatternMatches(pattern, stringMap))
	})
}

func TestMountPointAttachmentPattern(t *testing.T) {
	pattern := MountPointAttachmentPattern{}
	att := MountPointAttachment{}
	require.Equal(t, true, mountPointAttachmentPatternMatches(pattern, att))
	att = MountPointAttachment{EffectiveSource: "/new_dir"}
	require.Equal(t, true, mountPointAttachmentPatternMatches(pattern, att))
	att = MountPointAttachment{Consistency: "delegated"}
	require.Equal(t, true, mountPointAttachmentPatternMatches(pattern, att))

	pattern = MountPointAttachmentPattern{
		EffectiveSource: StringPattern{Exactly: "/new_dir"},
	}
	att = MountPointAttachment{}
	require.Equal(t, false, mountPointAttachmentPatternMatches(pattern, att))
	att = MountPointAttachment{EffectiveSource: "/new_dir"}
	require.Equal(t, true, mountPointAttachmentPatternMatches(pattern, att))
	att = MountPointAttachment{Consistency: "delegated"}
	require.Equal(t, false, mountPointAttachmentPatternMatches(pattern, att))

	delegated := mount.ConsistencyDelegated
	pattern = MountPointAttachmentPattern{
		Consistency: &delegated,
	}
	att = MountPointAttachment{}
	require.Equal(t, false, mountPointAttachmentPatternMatches(pattern, att))
	att = MountPointAttachment{EffectiveSource: "/new_dir"}
	require.Equal(t, false, mountPointAttachmentPatternMatches(pattern, att))
	att = MountPointAttachment{Consistency: mount.ConsistencyDelegated}
	require.Equal(t, true, mountPointAttachmentPatternMatches(pattern, att))
}

func TestAppliedPluginPattern(t *testing.T) {
	pattern := AppliedPluginPattern{}
	plugin := AppliedPlugin{}
	require.Equal(t, true, appliedPluginPatternMatches(pattern, plugin))
	plugin = AppliedPlugin{Name: "plugin"}
	require.Equal(t, true, appliedPluginPatternMatches(pattern, plugin))
	plugin = AppliedPlugin{
		MountPoint: MountPointAttachment{EffectiveSource: "/new/dir"},
	}
	require.Equal(t, true, appliedPluginPatternMatches(pattern, plugin))

	pattern = AppliedPluginPattern{
		Name: StringPattern{Exactly: "plugin"},
	}
	plugin = AppliedPlugin{}
	require.Equal(t, false, appliedPluginPatternMatches(pattern, plugin))
	plugin = AppliedPlugin{Name: "plugin"}
	require.Equal(t, true, appliedPluginPatternMatches(pattern, plugin))
	plugin = AppliedPlugin{
		MountPoint: MountPointAttachment{EffectiveSource: "/new/dir"},
	}
	require.Equal(t, false, appliedPluginPatternMatches(pattern, plugin))

	pattern = AppliedPluginPattern{
		MountPoint: MountPointAttachmentPattern{
			EffectiveSource: StringPattern{PathPrefix: "/new"},
		},
	}
	plugin = AppliedPlugin{}
	require.Equal(t, false, appliedPluginPatternMatches(pattern, plugin))
	plugin = AppliedPlugin{Name: "plugin"}
	require.Equal(t, false, appliedPluginPatternMatches(pattern, plugin))
	plugin = AppliedPlugin{
		MountPoint: MountPointAttachment{EffectiveSource: "/new/dir"},
	}
	require.Equal(t, true, appliedPluginPatternMatches(pattern, plugin))
}

func testAppliedPluginsPatternInverse(pattern AppliedPluginsPattern, f func(pattern AppliedPluginsPattern, tru, fals bool)) {
	f(pattern, true, false)
	pattern.Not = true
	f(pattern, false, true)
}

func TestAppliedPluginsPatternExists(t *testing.T) {
	testAppliedPluginsPatternInverse(AppliedPluginsPattern{
		Exists: []AppliedPluginPattern{
			{Name: StringPattern{Exactly: "plugin0"}},
		},
	}, func(pattern AppliedPluginsPattern, tru, fals bool) {
		list := []AppliedPlugin{}
		require.Equal(t, fals, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin0"},
			{Name: "plugin1"},
		}
		require.Equal(t, tru, appliedPluginsPatternMatches(pattern, list))
	})

	testAppliedPluginsPatternInverse(AppliedPluginsPattern{
		Exists: []AppliedPluginPattern{
			{Name: StringPattern{Exactly: "plugin0"}},
			{Name: StringPattern{Exactly: "plugin1"}},
		},
	}, func(pattern AppliedPluginsPattern, tru, fals bool) {
		list := []AppliedPlugin{}
		require.Equal(t, fals, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin0"},
		}
		require.Equal(t, false, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin1"},
		}
		require.Equal(t, false, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin0"},
			{Name: "plugin1"},
		}
		require.Equal(t, tru, appliedPluginsPatternMatches(pattern, list))
	})
}

func TestAppliedPluginsPatternAll(t *testing.T) {
	testAppliedPluginsPatternInverse(AppliedPluginsPattern{
		All: []AppliedPluginPattern{
			{Name: StringPattern{Exactly: "plugin0"}},
		},
	}, func(pattern AppliedPluginsPattern, tru, fals bool) {
		list := []AppliedPlugin{}
		require.Equal(t, tru, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin0"},
			{Name: "plugin1"},
		}
		require.Equal(t, fals, appliedPluginsPatternMatches(pattern, list))
	})

	testAppliedPluginsPatternInverse(AppliedPluginsPattern{
		All: []AppliedPluginPattern{
			{Name: StringPattern{Suffix: "_"}},
			{Name: StringPattern{Prefix: "p"}},
		},
	}, func(pattern AppliedPluginsPattern, tru, fals bool) {
		list := []AppliedPlugin{}
		require.Equal(t, tru, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin0"},
			{Name: "plugin1"},
		}
		require.Equal(t, false, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin0_"},
			{Name: "plugin1_"},
		}
		require.Equal(t, tru, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin0_"},
			{Name: "plugin1"},
		}
		require.Equal(t, false, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin0_"},
			{Name: "_plugin1"},
		}
		require.Equal(t, fals, appliedPluginsPatternMatches(pattern, list))
	})
}

func TestAppliedPluginsPatternAnySequence(t *testing.T) {
	testAppliedPluginsPatternInverse(AppliedPluginsPattern{
		AnySequence: []AppliedPluginPattern{
			{Name: StringPattern{Exactly: "plugin1"}},
			{Name: StringPattern{Exactly: "plugin2"}},
		},
	}, func(pattern AppliedPluginsPattern, tru, fals bool) {
		list := []AppliedPlugin{}
		require.Equal(t, fals, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin1"},
		}
		require.Equal(t, fals, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin1"},
			{Name: "plugin2"},
		}
		require.Equal(t, tru, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin0"},
			{Name: "plugin1"},
			{Name: "plugin2"},
		}
		require.Equal(t, tru, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin1"},
			{Name: "plugin2"},
			{Name: "plugin3"},
		}
		require.Equal(t, tru, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin0"},
			{Name: "plugin1"},
			{Name: "plugin2"},
			{Name: "plugin3"},
		}
		require.Equal(t, tru, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin1"},
			{Name: "plugin1"},
			{Name: "plugin2"},
		}
		require.Equal(t, tru, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin1"},
			{Name: "plugin0"},
			{Name: "plugin2"},
		}
		require.Equal(t, fals, appliedPluginsPatternMatches(pattern, list))
	})
}

func TestAppliedPluginsPatternTopSequence(t *testing.T) {
	testAppliedPluginsPatternInverse(AppliedPluginsPattern{
		TopSequence: []AppliedPluginPattern{
			{Name: StringPattern{Exactly: "plugin1"}},
			{Name: StringPattern{Exactly: "plugin2"}},
		},
	}, func(pattern AppliedPluginsPattern, tru, fals bool) {
		list := []AppliedPlugin{}
		require.Equal(t, fals, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin1"},
		}
		require.Equal(t, fals, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin1"},
			{Name: "plugin2"},
		}
		require.Equal(t, tru, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin0"},
			{Name: "plugin1"},
			{Name: "plugin2"},
		}
		require.Equal(t, fals, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin1"},
			{Name: "plugin2"},
			{Name: "plugin3"},
		}
		require.Equal(t, tru, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin1"},
			{Name: "plugin1"},
			{Name: "plugin2"},
		}
		require.Equal(t, fals, appliedPluginsPatternMatches(pattern, list))
	})
}

func TestAppliedPluginsPatternBottomSequence(t *testing.T) {
	testAppliedPluginsPatternInverse(AppliedPluginsPattern{
		BottomSequence: []AppliedPluginPattern{
			{Name: StringPattern{Exactly: "plugin1"}},
			{Name: StringPattern{Exactly: "plugin2"}},
		},
	}, func(pattern AppliedPluginsPattern, tru, fals bool) {
		list := []AppliedPlugin{}
		require.Equal(t, fals, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin1"},
		}
		require.Equal(t, fals, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin1"},
			{Name: "plugin2"},
		}
		require.Equal(t, tru, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin0"},
			{Name: "plugin1"},
			{Name: "plugin2"},
		}
		require.Equal(t, tru, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin1"},
			{Name: "plugin2"},
			{Name: "plugin3"},
		}
		require.Equal(t, fals, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin1"},
			{Name: "plugin1"},
			{Name: "plugin2"},
		}
		require.Equal(t, tru, appliedPluginsPatternMatches(pattern, list))
	})
}

func TestAppliedPluginsPatternRelativeOrder(t *testing.T) {
	testAppliedPluginsPatternInverse(AppliedPluginsPattern{
		RelativeOrder: []AppliedPluginPattern{
			{Name: StringPattern{Exactly: "plugin1"}},
			{Name: StringPattern{Exactly: "plugin2"}},
		},
	}, func(pattern AppliedPluginsPattern, tru, fals bool) {
		list := []AppliedPlugin{}
		require.Equal(t, fals, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin1"},
		}
		require.Equal(t, fals, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin1"},
			{Name: "plugin2"},
		}
		require.Equal(t, tru, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin0"},
			{Name: "plugin1"},
			{Name: "plugin2"},
		}
		require.Equal(t, tru, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin1"},
			{Name: "plugin2"},
			{Name: "plugin3"},
		}
		require.Equal(t, tru, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin1"},
			{Name: "plugin0"},
			{Name: "plugin2"},
		}
		require.Equal(t, tru, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin0"},
			{Name: "plugin1"},
			{Name: "plugin0"},
			{Name: "plugin2"},
		}
		require.Equal(t, tru, appliedPluginsPatternMatches(pattern, list))
		list = []AppliedPlugin{
			{Name: "plugin1"},
			{Name: "plugin0"},
			{Name: "plugin2"},
			{Name: "plugin0"},
		}
		require.Equal(t, tru, appliedPluginsPatternMatches(pattern, list))
	})
}

func TestPattern(t *testing.T) {
	mountpoint := &MountPoint{
		EffectiveSource: "/src",
		Source:          "/src",
		Destination:     "/mnt/pt",
		ReadOnly:        true,
		Name:            "MyVolume",
		Driver:          "local",
		Type:            TypeVolume,
		Mode:            "o=bind,foo=bar",
		Propagation:     mount.PropagationShared,
		ID:              "0123456789abcdef",

		AppliedPlugins: []AppliedPlugin{
			{Name: "mountPointPlugin0"},
			{Name: "mountPointPlugin1"},
		},

		Consistency: mount.ConsistencyCached,
		Labels: map[string]string{
			"label0": "value",
			"label1": "",
		},

		DriverOptions: map[string]string{
			"opt0": "x",
			"opt1": "y",
		},
		Scope: LocalScope,
	}

	pattern := MountPointPattern{}
	require.Equal(t, true, PatternMatches(pattern, mountpoint))
	pattern = MountPointPattern{
		EffectiveSource: &StringPattern{Exactly: "/src"},
	}
	require.Equal(t, true, PatternMatches(pattern, mountpoint))
	pattern = MountPointPattern{
		EffectiveSource: &StringPattern{Not: true, Exactly: "/src"},
	}
	require.Equal(t, false, PatternMatches(pattern, mountpoint))
	pattern = MountPointPattern{
		Source: &StringPattern{Exactly: "/src"},
	}
	require.Equal(t, true, PatternMatches(pattern, mountpoint))
	pattern = MountPointPattern{
		Source: &StringPattern{Not: true, Exactly: "/src"},
	}
	require.Equal(t, false, PatternMatches(pattern, mountpoint))
	pattern = MountPointPattern{
		Destination: &StringPattern{PathPrefix: "/mnt"},
	}
	require.Equal(t, true, PatternMatches(pattern, mountpoint))
	pattern = MountPointPattern{
		Destination: &StringPattern{Not: true, PathPrefix: "/mnt"},
	}
	require.Equal(t, false, PatternMatches(pattern, mountpoint))
	tru := true
	pattern = MountPointPattern{
		ReadOnly: &tru,
	}
	require.Equal(t, true, PatternMatches(pattern, mountpoint))
	fals := false
	pattern = MountPointPattern{
		ReadOnly: &fals,
	}
	require.Equal(t, false, PatternMatches(pattern, mountpoint))
	pattern = MountPointPattern{
		Name: &StringPattern{Exactly: "MyVolume"},
	}
	require.Equal(t, true, PatternMatches(pattern, mountpoint))
	pattern = MountPointPattern{
		Name: &StringPattern{Not: true, Exactly: "MyVolume"},
	}
	require.Equal(t, false, PatternMatches(pattern, mountpoint))
	pattern = MountPointPattern{
		Driver: &StringPattern{Exactly: "local"},
	}
	require.Equal(t, true, PatternMatches(pattern, mountpoint))
	pattern = MountPointPattern{
		Driver: &StringPattern{Not: true, Exactly: "local"},
	}
	require.Equal(t, false, PatternMatches(pattern, mountpoint))
	volume := TypeVolume
	pattern = MountPointPattern{
		Type: &volume,
	}
	require.Equal(t, true, PatternMatches(pattern, mountpoint))
	bind := TypeBind
	pattern = MountPointPattern{
		Type: &bind,
	}
	require.Equal(t, false, PatternMatches(pattern, mountpoint))
	pattern = MountPointPattern{
		Mode: &StringPattern{Contains: "o=bind"},
	}
	require.Equal(t, true, PatternMatches(pattern, mountpoint))
	pattern = MountPointPattern{
		Mode: &StringPattern{Not: true, Contains: "o=bind"},
	}
	require.Equal(t, false, PatternMatches(pattern, mountpoint))
	propagationShared := mount.PropagationShared
	pattern = MountPointPattern{
		Propagation: &propagationShared,
	}
	require.Equal(t, true, PatternMatches(pattern, mountpoint))
	propagationSlave := mount.PropagationSlave
	pattern = MountPointPattern{
		Propagation: &propagationSlave,
	}
	require.Equal(t, false, PatternMatches(pattern, mountpoint))
	pattern = MountPointPattern{
		ID: &StringPattern{Exactly: "0123456789abcdef"},
	}
	require.Equal(t, true, PatternMatches(pattern, mountpoint))
	pattern = MountPointPattern{
		ID: &StringPattern{Not: true, Exactly: "0123456789abcdef"},
	}
	require.Equal(t, false, PatternMatches(pattern, mountpoint))

	pattern = MountPointPattern{
		AppliedPlugins: &AppliedPluginsPattern{
			Exists: []AppliedPluginPattern{
				{Name: StringPattern{Exactly: "mountPointPlugin0"}},
			},
		},
	}
	require.Equal(t, true, PatternMatches(pattern, mountpoint))
	pattern = MountPointPattern{
		AppliedPlugins: &AppliedPluginsPattern{
			Not: true,
			Exists: []AppliedPluginPattern{
				{Name: StringPattern{Exactly: "mountPointPlugin0"}},
			},
		},
	}
	require.Equal(t, false, PatternMatches(pattern, mountpoint))

	cached := mount.ConsistencyCached
	pattern = MountPointPattern{
		Consistency: &cached,
	}
	require.Equal(t, true, PatternMatches(pattern, mountpoint))
	delegated := mount.ConsistencyDelegated
	pattern = MountPointPattern{
		Consistency: &delegated,
	}
	require.Equal(t, false, PatternMatches(pattern, mountpoint))
	pattern = MountPointPattern{
		Labels: &StringMapPattern{
			Exists: map[StringPattern]*StringPattern{
				{Exactly: "label0"}: nil,
			},
		},
	}
	require.Equal(t, true, PatternMatches(pattern, mountpoint))
	pattern = MountPointPattern{
		Labels: &StringMapPattern{
			Not: true,
			Exists: map[StringPattern]*StringPattern{
				{Exactly: "label0"}: nil,
			},
		},
	}
	require.Equal(t, false, PatternMatches(pattern, mountpoint))

	pattern = MountPointPattern{
		DriverOptions: &StringMapPattern{
			Exists: map[StringPattern]*StringPattern{
				{Exactly: "opt0"}: nil,
			},
		},
	}
	require.Equal(t, true, PatternMatches(pattern, mountpoint))
	pattern = MountPointPattern{
		DriverOptions: &StringMapPattern{
			Not: true,
			Exists: map[StringPattern]*StringPattern{
				{Exactly: "opt0"}: nil,
			},
		},
	}
	require.Equal(t, false, PatternMatches(pattern, mountpoint))
	localScope := LocalScope
	pattern = MountPointPattern{
		Scope: &localScope,
	}
	require.Equal(t, true, PatternMatches(pattern, mountpoint))
	globalScope := GlobalScope
	pattern = MountPointPattern{
		Scope: &globalScope,
	}
	require.Equal(t, false, PatternMatches(pattern, mountpoint))
}
