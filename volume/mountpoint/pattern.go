package mountpoint

import (
	"path/filepath"
	"strings"
)

// PatternMatches determines if a pattern matches a mount point
// description. Patterns are conjunctions and a higher-level routine
// must implement disjunction.
func PatternMatches(pattern MountPointPattern, mount *MountPoint) bool {
	for _, pattern := range pattern.EffectiveSource {
		if !stringPatternMatches(pattern, mount.EffectiveSource) {
			return false
		}
	}

	for _, pattern := range pattern.Source {
		if !stringPatternMatches(pattern, mount.Source) {
			return false
		}
	}

	for _, pattern := range pattern.Destination {
		if !stringPatternMatches(pattern, mount.Destination) {
			return false
		}
	}

	if pattern.ReadOnly != nil && *pattern.ReadOnly != mount.ReadOnly {
		return false
	}

	for _, pattern := range pattern.Name {
		if !stringPatternMatches(pattern, mount.Name) {
			return false
		}
	}

	for _, pattern := range pattern.Driver {
		if !stringPatternMatches(pattern, mount.Driver) {
			return false
		}
	}

	if pattern.Type != nil && *pattern.Type != mount.Type {
		return false
	}

	for _, pattern := range pattern.Mode {
		if !stringPatternMatches(pattern, mount.Mode) {
			return false
		}
	}

	if pattern.Propagation != nil && *pattern.Propagation != mount.Propagation {
		return false
	}

	for _, pattern := range pattern.ID {
		if !stringPatternMatches(pattern, mount.ID) {
			return false
		}
	}

	if pattern.AppliedPlugins != nil && !appliedPluginsPatternMatches(*pattern.AppliedPlugins, mount.AppliedPlugins) {
		return false
	}

	if pattern.Consistency != nil && *pattern.Consistency != mount.Consistency {
		return false
	}

	for _, pattern := range pattern.Labels {
		if !stringMapPatternMatches(pattern, mount.Labels) {
			return false
		}
	}

	for _, pattern := range pattern.DriverOptions {
		if !stringMapPatternMatches(pattern, mount.DriverOptions) {
			return false
		}
	}

	if pattern.Scope != nil && *pattern.Scope != mount.Scope {
		return false
	}

	return true
}

func appliedPluginsPatternMatches(pattern AppliedPluginsPattern, appliedPlugins []AppliedPlugin) bool {

	if !pluginsExist(pattern.Exists, appliedPlugins, false) {
		return false
	}
	if !pluginsExist(pattern.NotExists, appliedPlugins, true) {
		return false
	}

	if !pluginsAll(pattern.All, appliedPlugins, false) {
		return false
	}
	if !pluginsAll(pattern.NotAll, appliedPlugins, true) {
		return false
	}

	if !pluginsAnySequence(pattern.AnySequence, appliedPlugins, false) {
		return false
	}
	if !pluginsAnySequence(pattern.NotAnySequence, appliedPlugins, true) {
		return false
	}

	if !pluginsTopSequence(pattern.TopSequence, appliedPlugins, false) {
		return false
	}
	if !pluginsTopSequence(pattern.NotTopSequence, appliedPlugins, true) {
		return false
	}

	if !pluginsBottomSequence(pattern.BottomSequence, appliedPlugins, false) {
		return false
	}
	if !pluginsBottomSequence(pattern.NotBottomSequence, appliedPlugins, true) {
		return false
	}

	if !pluginsRelativeOrder(pattern.RelativeOrder, appliedPlugins, false) {
		return false
	}
	if !pluginsRelativeOrder(pattern.NotRelativeOrder, appliedPlugins, true) {
		return false
	}

	return true
}

func pluginsExist(patterns []AppliedPluginPattern, plugins []AppliedPlugin, not bool) bool {
	for _, pluginPattern := range patterns {
		matched := false
		for _, plugin := range plugins {
			if appliedPluginPatternMatches(pluginPattern, plugin) {
				matched = true
				break
			}
		}

		if matched == not {
			return false
		}
	}

	return true
}

func pluginsAll(patterns []AppliedPluginPattern, plugins []AppliedPlugin, not bool) bool {
	for _, pluginPattern := range patterns {
		matched := true
		for _, plugin := range plugins {
			if !appliedPluginPatternMatches(pluginPattern, plugin) {
				matched = false
				break
			}
		}

		if matched == not {
			return false
		}
	}

	return true
}

func pluginsAnySequence(patterns []AppliedPluginPattern, plugins []AppliedPlugin, not bool) bool {
	anySequenceCount := len(patterns)
	appliedPluginCount := len(plugins)
	if anySequenceCount > 0 {
		if anySequenceCount <= appliedPluginCount {
			found := false
			for i := 0; i <= (appliedPluginCount - anySequenceCount); i++ {
				matched := true
				for j, pluginPattern := range patterns {
					if !appliedPluginPatternMatches(pluginPattern, plugins[i+j]) {
						matched = false
						break
					}
				}
				if matched {
					found = true
					break
				}
			}
			if found == not {
				return false
			}
		} else if !not {
			return false
		}
	}

	return true
}

func pluginsTopSequence(patterns []AppliedPluginPattern, plugins []AppliedPlugin, not bool) bool {
	topSequenceCount := len(patterns)
	appliedPluginCount := len(plugins)
	if topSequenceCount > 0 {
		if topSequenceCount <= appliedPluginCount {
			matched := true
			for i, pluginPattern := range patterns {
				if !appliedPluginPatternMatches(pluginPattern, plugins[i]) {
					matched = false
					break
				}
			}
			if matched == not {
				return false
			}
		} else if !not {
			return false
		}
	}

	return true
}

func pluginsBottomSequence(patterns []AppliedPluginPattern, plugins []AppliedPlugin, not bool) bool {
	bottomSequenceCount := len(patterns)
	appliedPluginCount := len(plugins)
	if bottomSequenceCount > 0 {
		if bottomSequenceCount <= appliedPluginCount {
			matched := true
			start := appliedPluginCount - bottomSequenceCount
			for i, pluginPattern := range patterns {
				if !appliedPluginPatternMatches(pluginPattern, plugins[start+i]) {
					matched = false
					break
				}
			}
			if matched == not {
				return false
			}
		} else if !not {
			return false
		}
	}

	return true
}

func pluginsRelativeOrder(patterns []AppliedPluginPattern, plugins []AppliedPlugin, not bool) bool {
	relativeOrderCount := len(patterns)
	appliedPluginCount := len(plugins)
	if relativeOrderCount > 0 {
		if relativeOrderCount <= appliedPluginCount {
			remainingPatterns := patterns
			for _, plugin := range plugins {
				if len(remainingPatterns) == 0 {
					break
				}

				if appliedPluginPatternMatches(remainingPatterns[0], plugin) {
					remainingPatterns = remainingPatterns[1:]
				}
			}
			if (len(remainingPatterns) == 0) == not {
				return false
			}
		} else if !not {
			return false
		}
	}

	return true
}

func appliedPluginPatternMatches(pattern AppliedPluginPattern, appliedPlugin AppliedPlugin) bool {
	for _, spattern := range pattern.Name {
		if !stringPatternMatches(spattern, appliedPlugin.Name) {
			return false
		}
	}

	if !mountPointAttachmentPatternMatches(pattern.MountPoint, appliedPlugin.MountPoint) {
		return false
	}

	return true
}

func mountPointAttachmentPatternMatches(pattern MountPointAttachmentPattern, attachment MountPointAttachment) bool {

	for _, pattern := range pattern.EffectiveSource {
		if !stringPatternMatches(pattern, attachment.EffectiveSource) {
			return false
		}
	}

	if pattern.Consistency != nil && *pattern.Consistency != attachment.Consistency {
		return false
	}

	return true
}

func stringMapPatternMatches(pattern StringMapPattern, stringMap map[string]string) bool {

	// dsheets: These loops could almost certainly be fused but
	// reasoning about correctness would likely suffer. I don't think
	// patterns or maps will typically be big enough for the potential
	// constant (3x?) performance improvement to matter.

	for keyPattern, valuePatternOpt := range pattern.Exists {
		matched := false
		for key, value := range stringMap {
			if stringPatternMatches(keyPattern, key) {
				if valuePatternOpt == nil || stringPatternMatches(*valuePatternOpt, value) {
					matched = true
					break
				}
			}
		}

		if matched == pattern.Not {
			return false
		}
	}

	for keyPattern, valuePatternOpt := range pattern.All {
		matched := true
		for key, value := range stringMap {
			if stringPatternMatches(keyPattern, key) {
				if valuePatternOpt != nil && !stringPatternMatches(*valuePatternOpt, value) {
					matched = false
					break
				}
			} else if valuePatternOpt == nil {
				matched = false
				break
			}
		}

		if matched == pattern.Not {
			return false
		}
	}

	return true
}

func stringPatternMatches(pattern StringPattern, string string) bool {
	if pattern.Empty && (len(string) == 0) == pattern.Not {
		return false
	}

	if pattern.Prefix != "" && strings.HasPrefix(string, pattern.Prefix) == pattern.Not {
		return false
	}

	if pattern.PathPrefix != "" {
		cleanPath := filepath.Clean(string)
		cleanPattern := filepath.Clean(pattern.PathPrefix)
		patternLen := len(cleanPattern)

		matched := strings.HasPrefix(cleanPath, cleanPattern)
		if matched && cleanPattern[patternLen-1] != '/' {
			if len(cleanPath) > patternLen && cleanPath[patternLen] != '/' {
				matched = false
			}
		}

		if matched == pattern.Not {
			return false
		}
	}

	if pattern.Suffix != "" && strings.HasSuffix(string, pattern.Suffix) == pattern.Not {
		return false
	}

	if pattern.Exactly != "" && (pattern.Exactly == string) == pattern.Not {
		return false
	}

	if pattern.Contains != "" && strings.Contains(string, pattern.Contains) == pattern.Not {
		return false
	}

	return true
}
