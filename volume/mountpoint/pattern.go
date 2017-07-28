package mountpoint

import (
	"path/filepath"
	"strings"
)

// PatternMatches determines if a pattern matches a mount point
// description. Patterns are conjunctions and a higher-level routine
// must implement disjunction.
func PatternMatches(pattern MountPointPattern, mount *MountPoint) bool {
	if pattern.EffectiveSource != nil && !stringPatternMatches(*pattern.EffectiveSource, mount.EffectiveSource) {
		return false
	}

	if pattern.Source != nil && !stringPatternMatches(*pattern.Source, mount.Source) {
		return false
	}

	if pattern.Destination != nil && !stringPatternMatches(*pattern.Destination, mount.Destination) {
		return false
	}

	if pattern.ReadOnly != nil && *pattern.ReadOnly != mount.ReadOnly {
		return false
	}

	if pattern.Name != nil && !stringPatternMatches(*pattern.Name, mount.Name) {
		return false
	}

	if pattern.Driver != nil && !stringPatternMatches(*pattern.Driver, mount.Driver) {
		return false
	}

	if pattern.Type != nil && *pattern.Type != mount.Type {
		return false
	}

	if pattern.Mode != nil && !stringPatternMatches(*pattern.Mode, mount.Mode) {
		return false
	}

	if pattern.Propagation != nil && *pattern.Propagation != mount.Propagation {
		return false
	}

	if pattern.ID != nil && !stringPatternMatches(*pattern.ID, mount.ID) {
		return false
	}

	if pattern.AppliedPlugins != nil && !appliedPluginsPatternMatches(*pattern.AppliedPlugins, mount.AppliedPlugins) {
		return false
	}

	if pattern.Consistency != nil && *pattern.Consistency != mount.Consistency {
		return false
	}

	if pattern.Labels != nil && !stringMapPatternMatches(*pattern.Labels, mount.Labels) {
		return false
	}

	if pattern.DriverOptions != nil && !stringMapPatternMatches(*pattern.DriverOptions, mount.DriverOptions) {
		return false
	}

	if pattern.Scope != nil && *pattern.Scope != mount.Scope {
		return false
	}

	return true
}

func appliedPluginsPatternMatches(pattern AppliedPluginsPattern, appliedPlugins []AppliedPlugin) bool {

	appliedPluginCount := len(appliedPlugins)

	// dsheets: These loops could almost certainly be fused but
	// reasoning about correctness would likely suffer. I don't think
	// patterns or plugin lists will typically be big enough for the
	// potential constant (3x?) performance improvement to matter.

	for _, pluginPattern := range pattern.Exists {
		matched := false
		for _, plugin := range appliedPlugins {
			if appliedPluginPatternMatches(pluginPattern, plugin) {
				matched = true
				break
			}
		}

		if matched == pattern.Not {
			return false
		}
	}

	for _, pluginPattern := range pattern.All {
		matched := true
		for _, plugin := range appliedPlugins {
			if !appliedPluginPatternMatches(pluginPattern, plugin) {
				matched = false
				break
			}
		}

		if matched == pattern.Not {
			return false
		}
	}

	anySequenceCount := len(pattern.AnySequence)
	if anySequenceCount > 0 {
		if anySequenceCount <= appliedPluginCount {
			found := false
			for i := 0; i <= (appliedPluginCount - anySequenceCount); i++ {
				matched := true
				for j, pluginPattern := range pattern.AnySequence {
					if !appliedPluginPatternMatches(pluginPattern, appliedPlugins[i+j]) {
						matched = false
						break
					}
				}
				if matched {
					found = true
					break
				}
			}
			if found == pattern.Not {
				return false
			}
		} else if !pattern.Not {
			return false
		}
	}

	topSequenceCount := len(pattern.TopSequence)
	if topSequenceCount > 0 {
		if topSequenceCount <= appliedPluginCount {
			matched := true
			for i, pluginPattern := range pattern.TopSequence {
				if !appliedPluginPatternMatches(pluginPattern, appliedPlugins[i]) {
					matched = false
					break
				}
			}
			if matched == pattern.Not {
				return false
			}
		} else if !pattern.Not {
			return false
		}
	}

	bottomSequenceCount := len(pattern.BottomSequence)
	if bottomSequenceCount > 0 {
		if bottomSequenceCount <= appliedPluginCount {
			matched := true
			start := appliedPluginCount - bottomSequenceCount
			for i, pluginPattern := range pattern.BottomSequence {
				if !appliedPluginPatternMatches(pluginPattern, appliedPlugins[start+i]) {
					matched = false
					break
				}
			}
			if matched == pattern.Not {
				return false
			}
		} else if !pattern.Not {
			return false
		}
	}

	relativeOrderCount := len(pattern.RelativeOrder)
	if relativeOrderCount > 0 {
		if relativeOrderCount <= appliedPluginCount {
			remainingPatterns := pattern.RelativeOrder
			for _, plugin := range appliedPlugins {
				if len(remainingPatterns) == 0 {
					break
				}

				if appliedPluginPatternMatches(remainingPatterns[0], plugin) {
					remainingPatterns = remainingPatterns[1:]
				}
			}
			if (len(remainingPatterns) == 0) == pattern.Not {
				return false
			}
		} else if !pattern.Not {
			return false
		}
	}

	return true
}

func appliedPluginPatternMatches(pattern AppliedPluginPattern, appliedPlugin AppliedPlugin) bool {
	if !stringPatternMatches(pattern.Name, appliedPlugin.Name) {
		return false
	}

	if !mountPointAttachmentPatternMatches(pattern.MountPoint, appliedPlugin.MountPoint) {
		return false
	}

	return true
}

func mountPointAttachmentPatternMatches(pattern MountPointAttachmentPattern, attachment MountPointAttachment) bool {

	if !stringPatternMatches(pattern.EffectiveSource, attachment.EffectiveSource) {
		return false
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
