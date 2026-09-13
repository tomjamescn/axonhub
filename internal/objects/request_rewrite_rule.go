package objects

import (
	"errors"
	"fmt"
	"slices"
)

var (
	// ErrEmptyRequestRewriteModel is returned when a rule has no model pattern.
	ErrEmptyRequestRewriteModel = errors.New("model pattern is required for request rewrite rules")
	// ErrEmptyRequestRewriteField is returned when a field map has no value mappings.
	ErrEmptyRequestRewriteField = errors.New("value mappings are required for request rewrite field")
	// ErrDuplicateRequestRewriteField is returned when a field path appears twice in one rule.
	ErrDuplicateRequestRewriteField = errors.New("duplicate field in request rewrite mappings")
)

// RequestRewriteValueMap maps one source field value to a replacement value.
// It mirrors the ReasoningEffortMapping{From,To} pattern. The first matching
// source value wins. An empty replacement value clears the field.
type RequestRewriteValueMap struct {
	// From is the original field value to match. An empty value matches an
	// absent/empty field value.
	From string `json:"from"`
	// To is the replacement value. An empty value clears the field.
	To string `json:"to"`
}

// RequestRewriteFieldMap holds the value mappings for a single request field.
type RequestRewriteFieldMap struct {
	// Path is the JSON path of the request body field to rewrite, e.g. "reasoning_effort".
	Path string `json:"path"`
	// Values maps original field values to replacement values.
	Values []RequestRewriteValueMap `json:"values"`
}

// ValidateFieldMaps validates field maps: paths must be non-empty, each field
// must have at least one value mapping, and each field path must appear at
// most once per rule.
func ValidateFieldMaps(fields []RequestRewriteFieldMap) error {
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		if field.Path == "" {
			return fmt.Errorf("%w: field path is required", ErrEmptyRequestRewriteField)
		}

		if len(field.Values) == 0 {
			return fmt.Errorf("%w: field %q", ErrEmptyRequestRewriteField, field.Path)
		}

		if _, ok := seen[field.Path]; ok {
			return fmt.Errorf("%w: field %q", ErrDuplicateRequestRewriteField, field.Path)
		}

		seen[field.Path] = struct{}{}
	}

	return nil
}

// ApplyFieldMapsTo maps a field value through the merged field maps. It
// returns the (possibly changed) value and whether any mapping matched.
func ApplyFieldMapsTo(fields []RequestRewriteFieldMap, path, value string) (string, bool) {
	for _, field := range fields {
		if field.Path != path {
			continue
		}

		for _, v := range field.Values {
			if v.From == value {
				return v.To, true
			}
		}
	}

	return value, false
}

// SortedFieldPaths returns the stable-sorted unique field paths of the maps.
func SortedFieldPaths(fields []RequestRewriteFieldMap) []string {
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		if field.Path == "" {
			continue
		}

		seen[field.Path] = struct{}{}
	}

	paths := make([]string, 0, len(seen))
	for p := range seen {
		paths = append(paths, p)
	}

	slices.Sort(paths)

	return paths
}
