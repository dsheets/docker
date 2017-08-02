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

	if !appliedMiddlewareStackPatternMatches(pattern.AppliedMiddleware, mount.AppliedMiddleware) {
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

func appliedMiddlewareStackPatternMatches(pattern AppliedMiddlewareStackPattern, appliedMiddleware []AppliedMiddleware) bool {

	if !middlewareExist(pattern.Exists, appliedMiddleware, false) {
		return false
	}
	if !middlewareExist(pattern.NotExists, appliedMiddleware, true) {
		return false
	}

	if !middlewareAll(pattern.All, appliedMiddleware, false) {
		return false
	}
	if !middlewareAll(pattern.NotAll, appliedMiddleware, true) {
		return false
	}

	if !middlewareAnySequence(pattern.AnySequence, appliedMiddleware, false) {
		return false
	}
	if !middlewareAnySequence(pattern.NotAnySequence, appliedMiddleware, true) {
		return false
	}

	if !middlewareTopSequence(pattern.TopSequence, appliedMiddleware, false) {
		return false
	}
	if !middlewareTopSequence(pattern.NotTopSequence, appliedMiddleware, true) {
		return false
	}

	if !middlewareBottomSequence(pattern.BottomSequence, appliedMiddleware, false) {
		return false
	}
	if !middlewareBottomSequence(pattern.NotBottomSequence, appliedMiddleware, true) {
		return false
	}

	if !middlewareRelativeOrder(pattern.RelativeOrder, appliedMiddleware, false) {
		return false
	}
	if !middlewareRelativeOrder(pattern.NotRelativeOrder, appliedMiddleware, true) {
		return false
	}

	return true
}

func middlewareExist(patterns []AppliedMiddlewarePattern, middleware []AppliedMiddleware, not bool) bool {
	for _, middlewarePattern := range patterns {
		matched := false
		for _, middleware := range middleware {
			if appliedMiddlewarePatternMatches(middlewarePattern, middleware) {
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

func middlewareAll(patterns []AppliedMiddlewarePattern, middleware []AppliedMiddleware, not bool) bool {
	for _, middlewarePattern := range patterns {
		matched := true
		for _, middleware := range middleware {
			if !appliedMiddlewarePatternMatches(middlewarePattern, middleware) {
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

func middlewareAnySequence(patterns []AppliedMiddlewarePattern, middleware []AppliedMiddleware, not bool) bool {
	anySequenceCount := len(patterns)
	appliedMiddlewareCount := len(middleware)
	if anySequenceCount > 0 {
		if anySequenceCount <= appliedMiddlewareCount {
			found := false
			for i := 0; i <= (appliedMiddlewareCount - anySequenceCount); i++ {
				matched := true
				for j, middlewarePattern := range patterns {
					if !appliedMiddlewarePatternMatches(middlewarePattern, middleware[i+j]) {
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

func middlewareTopSequence(patterns []AppliedMiddlewarePattern, middleware []AppliedMiddleware, not bool) bool {
	topSequenceCount := len(patterns)
	appliedMiddlewareCount := len(middleware)
	if topSequenceCount > 0 {
		if topSequenceCount <= appliedMiddlewareCount {
			matched := true
			for i, middlewarePattern := range patterns {
				if !appliedMiddlewarePatternMatches(middlewarePattern, middleware[i]) {
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

func middlewareBottomSequence(patterns []AppliedMiddlewarePattern, middleware []AppliedMiddleware, not bool) bool {
	bottomSequenceCount := len(patterns)
	appliedMiddlewareCount := len(middleware)
	if bottomSequenceCount > 0 {
		if bottomSequenceCount <= appliedMiddlewareCount {
			matched := true
			start := appliedMiddlewareCount - bottomSequenceCount
			for i, middlewarePattern := range patterns {
				if !appliedMiddlewarePatternMatches(middlewarePattern, middleware[start+i]) {
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

func middlewareRelativeOrder(patterns []AppliedMiddlewarePattern, middleware []AppliedMiddleware, not bool) bool {
	relativeOrderCount := len(patterns)
	appliedMiddlewareCount := len(middleware)
	if relativeOrderCount > 0 {
		if relativeOrderCount <= appliedMiddlewareCount {
			remainingPatterns := patterns
			for _, middleware := range middleware {
				if len(remainingPatterns) == 0 {
					break
				}

				if appliedMiddlewarePatternMatches(remainingPatterns[0], middleware) {
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

func appliedMiddlewarePatternMatches(pattern AppliedMiddlewarePattern, appliedMiddleware AppliedMiddleware) bool {
	for _, spattern := range pattern.Name {
		if !stringPatternMatches(spattern, appliedMiddleware.Name) {
			return false
		}
	}

	if !mountPointAttachmentPatternMatches(pattern.MountPoint, appliedMiddleware.MountPoint) {
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

	for _, keyValuePattern := range pattern.Exists {
		matched := false
		for key, value := range stringMap {
			if stringPatternMatches(keyValuePattern.Key, key) {
				if stringPatternMatches(keyValuePattern.Value, value) {
					matched = true
					break
				}
			}
		}

		if matched == pattern.Not {
			return false
		}
	}

	for _, keyValuePattern := range pattern.All {
		matched := true
		for key, value := range stringMap {
			if stringPatternMatches(keyValuePattern.Key, key) {
				if !stringPatternMatches(keyValuePattern.Value, value) {
					matched = false
					break
				}
			} else if stringPatternIsEmpty(keyValuePattern.Value) {
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

func stringPatternIsEmpty(p StringPattern) bool {
	return !p.Empty && p.Prefix == "" && p.PathPrefix == "" && p.Suffix == "" && p.Exactly == "" && p.Contains == ""
}
