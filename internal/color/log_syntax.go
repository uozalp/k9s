// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

package color

import (
	"bytes"
	"strings"
)

// LogSyntaxHighlighter provides fast syntax highlighting for log content using string matching.
type LogSyntaxHighlighter struct {
	errorKeywords []string
	warnKeywords  []string
	infoKeywords  []string
}

// NewLogSyntaxHighlighter creates a new log syntax highlighter with fast string matching.
func NewLogSyntaxHighlighter() *LogSyntaxHighlighter {
	return &LogSyntaxHighlighter{
		errorKeywords: []string{"ERROR", "error", "FATAL", "fatal", "PANIC", "panic", "Exception", "exception", "fail", "FAIL"},
		warnKeywords:  []string{"WARN", "warn", "WARNING", "warning"},
		infoKeywords:  []string{"INFO", "info"},
	}
}

// Highlight applies fast syntax highlighting to log content using string matching.
func (h *LogSyntaxHighlighter) Highlight(content string) string {
	// Fast string matching approach - much faster than regex

	// Check for error keywords first (most important)
	for _, keyword := range h.errorKeywords {
		if strings.Contains(content, keyword) {
			return Colorize(content, Red)
		}
	}

	// Check for warning keywords
	for _, keyword := range h.warnKeywords {
		if strings.Contains(content, keyword) {
			return Colorize(content, Yellow)
		}
	}

	// Check for info keywords
	for _, keyword := range h.infoKeywords {
		if strings.Contains(content, keyword) {
			return Colorize(content, Blue)
		}
	}

	// For testing - color lines with numbers in green to see if highlighting works at all
	if strings.ContainsAny(content, "0123456789") {
		return Colorize(content, Green)
	}

	// Default - no highlighting
	return content
}

// HighlightBytes applies syntax highlighting to log content as bytes.
func (h *LogSyntaxHighlighter) HighlightBytes(content []byte) []byte {
	// Fast byte-based approach to avoid string conversion

	// Check for error keywords
	for _, keyword := range h.errorKeywords {
		if bytes.Contains(content, []byte(keyword)) {
			return []byte(Colorize(string(content), Red))
		}
	}

	// Check for warning keywords
	for _, keyword := range h.warnKeywords {
		if bytes.Contains(content, []byte(keyword)) {
			return []byte(Colorize(string(content), Yellow))
		}
	}

	// Check for info keywords
	for _, keyword := range h.infoKeywords {
		if bytes.Contains(content, []byte(keyword)) {
			return []byte(Colorize(string(content), Blue))
		}
	}

	// For testing - color lines with numbers in green to see if highlighting works at all
	contentStr := string(content)
	if strings.ContainsAny(contentStr, "0123456789") {
		return []byte(Colorize(contentStr, Green))
	}

	// Default - no highlighting
	return content
}
