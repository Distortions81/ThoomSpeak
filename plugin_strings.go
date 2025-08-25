package main

import "strings"

// This file exposes small, easy-to-understand string helper functions to
// plugin scripts.  They wrap common operations from Go's standard library in
// a friendly form so macro authors do not need to understand the entire
// strings package.

// pluginIgnoreCase reports whether a and b are the same, ignoring letter case.
func pluginIgnoreCase(a, b string) bool { return strings.EqualFold(a, b) }

// pluginStartsWith reports whether s begins with prefix.
func pluginStartsWith(s, prefix string) bool { return strings.HasPrefix(s, prefix) }

// pluginEndsWith reports whether s ends with suffix.
func pluginEndsWith(s, suffix string) bool { return strings.HasSuffix(s, suffix) }

// pluginIncludes reports whether substr appears anywhere inside s.
func pluginIncludes(s, substr string) bool { return strings.Contains(s, substr) }

// pluginLower converts all letters in s to lower case.
func pluginLower(s string) string { return strings.ToLower(s) }

// pluginUpper converts all letters in s to upper case.
func pluginUpper(s string) string { return strings.ToUpper(s) }

// pluginTrim removes leading and trailing white space from s.
func pluginTrim(s string) string { return strings.TrimSpace(s) }

// pluginTrimStart removes prefix from the start of s if it is present.
func pluginTrimStart(s, prefix string) string { return strings.TrimPrefix(s, prefix) }

// pluginTrimEnd removes suffix from the end of s if it is present.
func pluginTrimEnd(s, suffix string) string { return strings.TrimSuffix(s, suffix) }

// pluginWords splits s around spaces and returns the separate words.
func pluginWords(s string) []string { return strings.Fields(s) }

// pluginJoin joins the text elements in parts inserting sep between each one.
func pluginJoin(parts []string, sep string) string { return strings.Join(parts, sep) }

// pluginReplace returns s with every instance of old replaced by new.
func pluginReplace(s, old, new string) string { return strings.ReplaceAll(s, old, new) }

// pluginSplit breaks s into pieces separated by sep.
func pluginSplit(s, sep string) []string { return strings.Split(s, sep) }
