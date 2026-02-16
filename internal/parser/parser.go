// Package parser extracts frontmatter, wikilinks, and tags from Markdown content.
package parser

import (
	"bytes"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	wikilinkRe = regexp.MustCompile(`\[\[(.*?)\]\]`)
	tagRe      = regexp.MustCompile(`(?:^|\s)#([A-Za-z][A-Za-z0-9_/-]*)`)
)

// Result holds the output of parsing a Markdown file.
type Result struct {
	Frontmatter map[string]interface{}
	Body        string
	Links       []string
	Tags        []string
	Title       string
}

// Parse extracts frontmatter, body, wikilinks, and tags from raw Markdown bytes.
func Parse(data []byte) (*Result, error) {
	fm, body, err := splitFrontmatter(data)
	if err != nil {
		return nil, err
	}

	links := extractLinks(body)
	tags := extractTags(body, fm)
	title := deriveTitle(fm, body)

	return &Result{
		Frontmatter: fm,
		Body:        body,
		Links:       links,
		Tags:        tags,
		Title:       title,
	}, nil
}

// splitFrontmatter separates YAML frontmatter (between leading --- delimiters)
// from the Markdown body. If no frontmatter is found the entire content is body.
func splitFrontmatter(data []byte) (map[string]interface{}, string, error) {
	const delim = "---"
	trimmed := bytes.TrimLeft(data, "\n\r")

	if !bytes.HasPrefix(trimmed, []byte(delim)) {
		return nil, string(data), nil
	}

	// Find end delimiter.
	rest := trimmed[len(delim):]
	idx := bytes.Index(rest, []byte("\n"+delim))
	if idx < 0 {
		// No closing delimiter — treat everything as body.
		return nil, string(data), nil
	}

	yamlBlock := rest[:idx]
	// Body starts after closing delimiter line.
	afterDelim := rest[idx+1+len(delim):]
	body := strings.TrimLeft(string(afterDelim), "\n\r")

	var fm map[string]interface{}
	if err := yaml.Unmarshal(yamlBlock, &fm); err != nil {
		// Invalid YAML — return body only, no error (spec: fallback).
		return nil, string(data), nil
	}

	return fm, body, nil
}

// extractLinks returns deduplicated wikilink targets, normalising aliases.
func extractLinks(body string) []string {
	matches := wikilinkRe.FindAllStringSubmatch(body, -1)
	seen := make(map[string]struct{}, len(matches))
	var out []string
	for _, m := range matches {
		raw := m[1]
		// Handle aliases: [[Target|Alias]] → Target.
		target := raw
		if i := strings.Index(raw, "|"); i >= 0 {
			target = raw[:i]
		}
		target = strings.TrimSpace(target)
		if target == "" {
			continue
		}
		if _, ok := seen[target]; ok {
			continue
		}
		seen[target] = struct{}{}
		out = append(out, target)
	}
	return out
}

// extractTags collects #tags from body and from frontmatter "tags" field.
func extractTags(body string, fm map[string]interface{}) []string {
	seen := make(map[string]struct{})
	var out []string

	// Tags from frontmatter.
	if fm != nil {
		if raw, ok := fm["tags"]; ok {
			switch v := raw.(type) {
			case []interface{}:
				for _, item := range v {
					if s, ok := item.(string); ok {
						s = strings.TrimSpace(s)
						if s != "" {
							if _, dup := seen[s]; !dup {
								seen[s] = struct{}{}
								out = append(out, s)
							}
						}
					}
				}
			}
		}
	}

	// Inline #tags from body.
	matches := tagRe.FindAllStringSubmatch(body, -1)
	for _, m := range matches {
		t := m[1]
		if _, dup := seen[t]; !dup {
			seen[t] = struct{}{}
			out = append(out, t)
		}
	}

	return out
}

// deriveTitle returns the frontmatter "title" if present, otherwise the first
// H1 heading, otherwise empty string.
func deriveTitle(fm map[string]interface{}, body string) string {
	if fm != nil {
		if t, ok := fm["title"]; ok {
			if s, ok := t.(string); ok && s != "" {
				return s
			}
		}
	}
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(trimmed[2:])
		}
	}
	return ""
}
