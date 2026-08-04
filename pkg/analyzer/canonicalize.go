package analyzer

import "regexp"

// canonicalizer applies a regex-based substitution to an exit message before
// signature hashing. Ordering is significant (e.g. timestamps must run before
// generic number collapsing so their digit groups do not get swept up).
type canonicalizer struct {
	name    string
	pattern *regexp.Regexp
	replace string
}

const maxCanonicalLength = 2000

var canonicalizers = []canonicalizer{
	{
		name:    "iso8601_timestamp",
		pattern: regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?`),
		replace: "<TS>",
	},
	{
		name:    "uuid",
		pattern: regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`),
		replace: "<UUID>",
	},
	{
		name:    "pid_tag",
		pattern: regexp.MustCompile(`\bpid=\d+`),
		replace: "pid=<PID>",
	},
	{
		name:    "url_query_string",
		pattern: regexp.MustCompile(`\?[^\s"'<>]+`),
		replace: "?<QS>",
	},
	{
		name:    "absolute_path",
		pattern: regexp.MustCompile(`(?:/[A-Za-z0-9_.\-]+){2,}`),
		replace: "<PATH>",
	},
	{
		name:    "hex_blob",
		pattern: regexp.MustCompile(`(?i)\b[0-9a-f]{8,}\b`),
		replace: "<HEX>",
	},
	{
		// No trailing \b: in Go regexp \b sits between \w and \W, so `15432ms`
		// has no boundary between digits and letters — we still want to
		// collapse the leading digit run.
		name:    "long_number",
		pattern: regexp.MustCompile(`\b\d{3,}`),
		replace: "<N>",
	},
}

// Canonicalize strips volatile fragments out of an exit message so that the
// same underlying failure produces a stable signature across runs. Returns an
// empty string for an empty input.
func Canonicalize(s string) string {
	if s == "" {
		return ""
	}
	for _, c := range canonicalizers {
		s = c.pattern.ReplaceAllString(s, c.replace)
	}
	if len(s) > maxCanonicalLength {
		s = s[:maxCanonicalLength]
	}
	return s
}
