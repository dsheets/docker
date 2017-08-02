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
		Exists: []StringMapKeyValuePattern{
			{Key: StringPattern{Exactly: "key"}},
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
		Exists: []StringMapKeyValuePattern{{
			Key:   StringPattern{Exactly: "key"},
			Value: StringPattern{Exactly: "value"},
		}},
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
		Exists: []StringMapKeyValuePattern{
			{Key: StringPattern{Exactly: "key"}},
			{Key: StringPattern{Exactly: "k2"}},
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
		Exists: []StringMapKeyValuePattern{
			{
				Key:   StringPattern{},
				Value: StringPattern{Prefix: "abc"},
			},
			{
				Key:   StringPattern{Exactly: "key"},
				Value: StringPattern{Suffix: "bcd"},
			},
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
		All: []StringMapKeyValuePattern{
			{Key: StringPattern{Prefix: "pre"}},
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
		All: []StringMapKeyValuePattern{
			{Key: StringPattern{Prefix: "pre"}},
			{Key: StringPattern{Suffix: "x"}},
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
		All: []StringMapKeyValuePattern{
			{
				Key:   StringPattern{Prefix: "key"},
				Value: StringPattern{Exactly: "value"},
			},
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
		All: []StringMapKeyValuePattern{
			{
				Key:   StringPattern{Prefix: "key"},
				Value: StringPattern{Prefix: "v"},
			},
			{
				Key:   StringPattern{Suffix: "_"},
				Value: StringPattern{Suffix: "e"},
			},
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
		EffectiveSource: []StringPattern{{Exactly: "/new_dir"}},
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

func TestAppliedMiddlewarePattern(t *testing.T) {
	pattern := AppliedMiddlewarePattern{}
	middleware := AppliedMiddleware{}
	require.Equal(t, true, appliedMiddlewarePatternMatches(pattern, middleware))
	middleware = AppliedMiddleware{Name: "plugin:plugin"}
	require.Equal(t, true, appliedMiddlewarePatternMatches(pattern, middleware))
	middleware = AppliedMiddleware{
		MountPoint: MountPointAttachment{EffectiveSource: "/new/dir"},
	}
	require.Equal(t, true, appliedMiddlewarePatternMatches(pattern, middleware))

	pattern = AppliedMiddlewarePattern{
		Name: []StringPattern{{Exactly: "plugin:plugin"}},
	}
	middleware = AppliedMiddleware{}
	require.Equal(t, false, appliedMiddlewarePatternMatches(pattern, middleware))
	middleware = AppliedMiddleware{Name: "plugin:plugin"}
	require.Equal(t, true, appliedMiddlewarePatternMatches(pattern, middleware))
	middleware = AppliedMiddleware{
		MountPoint: MountPointAttachment{EffectiveSource: "/new/dir"},
	}
	require.Equal(t, false, appliedMiddlewarePatternMatches(pattern, middleware))

	pattern = AppliedMiddlewarePattern{
		MountPoint: MountPointAttachmentPattern{
			EffectiveSource: []StringPattern{{PathPrefix: "/new"}},
		},
	}
	middleware = AppliedMiddleware{}
	require.Equal(t, false, appliedMiddlewarePatternMatches(pattern, middleware))
	middleware = AppliedMiddleware{Name: "plugin:plugin"}
	require.Equal(t, false, appliedMiddlewarePatternMatches(pattern, middleware))
	middleware = AppliedMiddleware{
		MountPoint: MountPointAttachment{EffectiveSource: "/new/dir"},
	}
	require.Equal(t, true, appliedMiddlewarePatternMatches(pattern, middleware))
}

func testAppliedMiddlewareStackPatternInverse(pattern AppliedMiddlewareStackPattern, f func(pattern AppliedMiddlewareStackPattern, tru, fals bool)) {
	f(pattern, true, false)
	pattern.NotExists = pattern.Exists
	pattern.Exists = []AppliedMiddlewarePattern{}
	pattern.NotAll = pattern.All
	pattern.All = []AppliedMiddlewarePattern{}
	pattern.NotAnySequence = pattern.AnySequence
	pattern.AnySequence = []AppliedMiddlewarePattern{}
	pattern.NotTopSequence = pattern.TopSequence
	pattern.TopSequence = []AppliedMiddlewarePattern{}
	pattern.NotBottomSequence = pattern.BottomSequence
	pattern.BottomSequence = []AppliedMiddlewarePattern{}
	pattern.NotRelativeOrder = pattern.RelativeOrder
	pattern.RelativeOrder = []AppliedMiddlewarePattern{}
	f(pattern, false, true)
}

func TestAppliedMiddlewareStackPatternExists(t *testing.T) {
	testAppliedMiddlewareStackPatternInverse(AppliedMiddlewareStackPattern{
		Exists: []AppliedMiddlewarePattern{
			{Name: []StringPattern{{Exactly: "plugin:plugin0"}}},
		},
	}, func(pattern AppliedMiddlewareStackPattern, tru, fals bool) {
		list := []AppliedMiddleware{}
		require.Equal(t, fals, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin0"},
		}
		require.Equal(t, tru, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin0"},
			{Name: "plugin:plugin1"},
		}
		require.Equal(t, tru, appliedMiddlewareStackPatternMatches(pattern, list))
	})

	testAppliedMiddlewareStackPatternInverse(AppliedMiddlewareStackPattern{
		Exists: []AppliedMiddlewarePattern{
			{Name: []StringPattern{{Exactly: "plugin:plugin0"}}},
			{Name: []StringPattern{{Exactly: "plugin:plugin1"}}},
		},
	}, func(pattern AppliedMiddlewareStackPattern, tru, fals bool) {
		list := []AppliedMiddleware{}
		require.Equal(t, fals, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin0"},
		}
		require.Equal(t, false, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin1"},
		}
		require.Equal(t, false, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin0"},
			{Name: "plugin:plugin1"},
		}
		require.Equal(t, tru, appliedMiddlewareStackPatternMatches(pattern, list))
	})
}

func TestAppliedMiddlewareStackPatternAll(t *testing.T) {
	testAppliedMiddlewareStackPatternInverse(AppliedMiddlewareStackPattern{
		All: []AppliedMiddlewarePattern{
			{Name: []StringPattern{{Exactly: "plugin:plugin0"}}},
		},
	}, func(pattern AppliedMiddlewareStackPattern, tru, fals bool) {
		list := []AppliedMiddleware{}
		require.Equal(t, tru, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin0"},
			{Name: "plugin:plugin1"},
		}
		require.Equal(t, fals, appliedMiddlewareStackPatternMatches(pattern, list))
	})

	testAppliedMiddlewareStackPatternInverse(AppliedMiddlewareStackPattern{
		All: []AppliedMiddlewarePattern{
			{Name: []StringPattern{{Suffix: "_"}}},
			{Name: []StringPattern{{Prefix: "plugin:p"}}},
		},
	}, func(pattern AppliedMiddlewareStackPattern, tru, fals bool) {
		list := []AppliedMiddleware{}
		require.Equal(t, tru, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin0"},
			{Name: "plugin:plugin1"},
		}
		require.Equal(t, false, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin0_"},
			{Name: "plugin:plugin1_"},
		}
		require.Equal(t, tru, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin0_"},
			{Name: "plugin:plugin1"},
		}
		require.Equal(t, false, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin0_"},
			{Name: "plugin:_plugin1"},
		}
		require.Equal(t, fals, appliedMiddlewareStackPatternMatches(pattern, list))
	})
}

func TestAppliedMiddlewareStackPatternAnySequence(t *testing.T) {
	testAppliedMiddlewareStackPatternInverse(AppliedMiddlewareStackPattern{
		AnySequence: []AppliedMiddlewarePattern{
			{Name: []StringPattern{{Exactly: "plugin:plugin1"}}},
			{Name: []StringPattern{{Exactly: "plugin:plugin2"}}},
		},
	}, func(pattern AppliedMiddlewareStackPattern, tru, fals bool) {
		list := []AppliedMiddleware{}
		require.Equal(t, fals, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin1"},
		}
		require.Equal(t, fals, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin1"},
			{Name: "plugin:plugin2"},
		}
		require.Equal(t, tru, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin0"},
			{Name: "plugin:plugin1"},
			{Name: "plugin:plugin2"},
		}
		require.Equal(t, tru, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin1"},
			{Name: "plugin:plugin2"},
			{Name: "plugin:plugin3"},
		}
		require.Equal(t, tru, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin0"},
			{Name: "plugin:plugin1"},
			{Name: "plugin:plugin2"},
			{Name: "plugin:plugin3"},
		}
		require.Equal(t, tru, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin1"},
			{Name: "plugin:plugin1"},
			{Name: "plugin:plugin2"},
		}
		require.Equal(t, tru, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin1"},
			{Name: "plugin:plugin0"},
			{Name: "plugin:plugin2"},
		}
		require.Equal(t, fals, appliedMiddlewareStackPatternMatches(pattern, list))
	})
}

func TestAppliedMiddlewareStackPatternTopSequence(t *testing.T) {
	testAppliedMiddlewareStackPatternInverse(AppliedMiddlewareStackPattern{
		TopSequence: []AppliedMiddlewarePattern{
			{Name: []StringPattern{{Exactly: "plugin:plugin1"}}},
			{Name: []StringPattern{{Exactly: "plugin:plugin2"}}},
		},
	}, func(pattern AppliedMiddlewareStackPattern, tru, fals bool) {
		list := []AppliedMiddleware{}
		require.Equal(t, fals, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin1"},
		}
		require.Equal(t, fals, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin1"},
			{Name: "plugin:plugin2"},
		}
		require.Equal(t, tru, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin0"},
			{Name: "plugin:plugin1"},
			{Name: "plugin:plugin2"},
		}
		require.Equal(t, fals, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin1"},
			{Name: "plugin:plugin2"},
			{Name: "plugin:plugin3"},
		}
		require.Equal(t, tru, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin1"},
			{Name: "plugin:plugin1"},
			{Name: "plugin:plugin2"},
		}
		require.Equal(t, fals, appliedMiddlewareStackPatternMatches(pattern, list))
	})
}

func TestAppliedMiddlewareStackPatternBottomSequence(t *testing.T) {
	testAppliedMiddlewareStackPatternInverse(AppliedMiddlewareStackPattern{
		BottomSequence: []AppliedMiddlewarePattern{
			{Name: []StringPattern{{Exactly: "plugin:plugin1"}}},
			{Name: []StringPattern{{Exactly: "plugin:plugin2"}}},
		},
	}, func(pattern AppliedMiddlewareStackPattern, tru, fals bool) {
		list := []AppliedMiddleware{}
		require.Equal(t, fals, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin1"},
		}
		require.Equal(t, fals, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin1"},
			{Name: "plugin:plugin2"},
		}
		require.Equal(t, tru, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin0"},
			{Name: "plugin:plugin1"},
			{Name: "plugin:plugin2"},
		}
		require.Equal(t, tru, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin1"},
			{Name: "plugin:plugin2"},
			{Name: "plugin:plugin3"},
		}
		require.Equal(t, fals, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin1"},
			{Name: "plugin:plugin1"},
			{Name: "plugin:plugin2"},
		}
		require.Equal(t, tru, appliedMiddlewareStackPatternMatches(pattern, list))
	})
}

func TestAppliedMiddlewareStackPatternRelativeOrder(t *testing.T) {
	testAppliedMiddlewareStackPatternInverse(AppliedMiddlewareStackPattern{
		RelativeOrder: []AppliedMiddlewarePattern{
			{Name: []StringPattern{{Exactly: "plugin:plugin1"}}},
			{Name: []StringPattern{{Exactly: "plugin:plugin2"}}},
		},
	}, func(pattern AppliedMiddlewareStackPattern, tru, fals bool) {
		list := []AppliedMiddleware{}
		require.Equal(t, fals, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin1"},
		}
		require.Equal(t, fals, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin1"},
			{Name: "plugin:plugin2"},
		}
		require.Equal(t, tru, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin0"},
			{Name: "plugin:plugin1"},
			{Name: "plugin:plugin2"},
		}
		require.Equal(t, tru, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin1"},
			{Name: "plugin:plugin2"},
			{Name: "plugin:plugin3"},
		}
		require.Equal(t, tru, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin1"},
			{Name: "plugin:plugin0"},
			{Name: "plugin:plugin2"},
		}
		require.Equal(t, tru, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin0"},
			{Name: "plugin:plugin1"},
			{Name: "plugin:plugin0"},
			{Name: "plugin:plugin2"},
		}
		require.Equal(t, tru, appliedMiddlewareStackPatternMatches(pattern, list))
		list = []AppliedMiddleware{
			{Name: "plugin:plugin1"},
			{Name: "plugin:plugin0"},
			{Name: "plugin:plugin2"},
			{Name: "plugin:plugin0"},
		}
		require.Equal(t, tru, appliedMiddlewareStackPatternMatches(pattern, list))
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

		AppliedMiddleware: []AppliedMiddleware{
			{Name: "plugin:mountPointPlugin0"},
			{Name: "plugin:mountPointPlugin1"},
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
		EffectiveSource: []StringPattern{{Exactly: "/src"}},
	}
	require.Equal(t, true, PatternMatches(pattern, mountpoint))
	pattern = MountPointPattern{
		EffectiveSource: []StringPattern{{Not: true, Exactly: "/src"}},
	}
	require.Equal(t, false, PatternMatches(pattern, mountpoint))
	pattern = MountPointPattern{
		Source: []StringPattern{{Exactly: "/src"}},
	}
	require.Equal(t, true, PatternMatches(pattern, mountpoint))
	pattern = MountPointPattern{
		Source: []StringPattern{{Not: true, Exactly: "/src"}},
	}
	require.Equal(t, false, PatternMatches(pattern, mountpoint))
	pattern = MountPointPattern{
		Destination: []StringPattern{{PathPrefix: "/mnt"}},
	}
	require.Equal(t, true, PatternMatches(pattern, mountpoint))
	pattern = MountPointPattern{
		Destination: []StringPattern{{Not: true, PathPrefix: "/mnt"}},
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
		Name: []StringPattern{{Exactly: "MyVolume"}},
	}
	require.Equal(t, true, PatternMatches(pattern, mountpoint))
	pattern = MountPointPattern{
		Name: []StringPattern{{Not: true, Exactly: "MyVolume"}},
	}
	require.Equal(t, false, PatternMatches(pattern, mountpoint))
	pattern = MountPointPattern{
		Driver: []StringPattern{{Exactly: "local"}},
	}
	require.Equal(t, true, PatternMatches(pattern, mountpoint))
	pattern = MountPointPattern{
		Driver: []StringPattern{{Not: true, Exactly: "local"}},
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
		Mode: []StringPattern{{Contains: "o=bind"}},
	}
	require.Equal(t, true, PatternMatches(pattern, mountpoint))
	pattern = MountPointPattern{
		Mode: []StringPattern{{Not: true, Contains: "o=bind"}},
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
		ID: []StringPattern{{Exactly: "0123456789abcdef"}},
	}
	require.Equal(t, true, PatternMatches(pattern, mountpoint))
	pattern = MountPointPattern{
		ID: []StringPattern{{Not: true, Exactly: "0123456789abcdef"}},
	}
	require.Equal(t, false, PatternMatches(pattern, mountpoint))

	pattern = MountPointPattern{
		AppliedMiddleware: AppliedMiddlewareStackPattern{
			Exists: []AppliedMiddlewarePattern{
				{Name: []StringPattern{{Exactly: "plugin:mountPointPlugin0"}}},
			},
		},
	}
	require.Equal(t, true, PatternMatches(pattern, mountpoint))
	pattern = MountPointPattern{
		AppliedMiddleware: AppliedMiddlewareStackPattern{
			NotExists: []AppliedMiddlewarePattern{{
				Name: []StringPattern{{Exactly: "plugin:mountPointPlugin0"}},
			}},
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
		Labels: []StringMapPattern{{
			Exists: []StringMapKeyValuePattern{
				{Key: StringPattern{Exactly: "label0"}},
			},
		}},
	}
	require.Equal(t, true, PatternMatches(pattern, mountpoint))
	pattern = MountPointPattern{
		Labels: []StringMapPattern{{
			Not: true,
			Exists: []StringMapKeyValuePattern{
				{Key: StringPattern{Exactly: "label0"}},
			},
		}},
	}
	require.Equal(t, false, PatternMatches(pattern, mountpoint))

	pattern = MountPointPattern{
		DriverOptions: []StringMapPattern{{
			Exists: []StringMapKeyValuePattern{
				{Key: StringPattern{Exactly: "opt0"}},
			},
		}},
	}
	require.Equal(t, true, PatternMatches(pattern, mountpoint))
	pattern = MountPointPattern{
		DriverOptions: []StringMapPattern{{
			Not: true,
			Exists: []StringMapKeyValuePattern{
				{Key: StringPattern{Exactly: "opt0"}},
			},
		}},
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
