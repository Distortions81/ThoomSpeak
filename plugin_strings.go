package main

import "strings"

func pluginIgnoreCase(a, b string) bool { return strings.EqualFold(a, b) }

func pluginStartsWith(s, prefix string) bool { return strings.HasPrefix(s, prefix) }

func pluginEndsWith(s, suffix string) bool { return strings.HasSuffix(s, suffix) }

func pluginIncludes(s, substr string) bool { return strings.Contains(s, substr) }

func pluginLower(s string) string { return strings.ToLower(s) }

func pluginUpper(s string) string { return strings.ToUpper(s) }

func pluginTrim(s string) string { return strings.TrimSpace(s) }

func pluginTrimStart(s, prefix string) string { return strings.TrimPrefix(s, prefix) }

func pluginTrimEnd(s, suffix string) string { return strings.TrimSuffix(s, suffix) }

func pluginWords(s string) []string { return strings.Fields(s) }

func pluginJoin(parts []string, sep string) string { return strings.Join(parts, sep) }
