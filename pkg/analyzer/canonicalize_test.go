package analyzer

import "testing"

func TestCanonicalize(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "empty",
			in:   "",
			want: "",
		},
		{
			name: "iso8601_timestamp",
			in:   "failure at 2026-03-26T14:57:10.123Z in handler",
			want: "failure at <TS> in handler",
		},
		{
			name: "iso8601_timestamp_with_offset",
			in:   "failed 2026-03-26T14:57:10+02:00",
			want: "failed <TS>",
		},
		{
			name: "uuid",
			in:   "dashboard 550e8400-e29b-41d4-a716-446655440000 not found",
			want: "dashboard <UUID> not found",
		},
		{
			name: "pid",
			in:   "process pid=12345 exited",
			want: "process pid=<PID> exited",
		},
		{
			name: "url_query_string",
			in:   "GET /api/folders?orgId=1&cacheBust=abc failed",
			want: "GET <PATH>?<QS> failed",
		},
		{
			name: "absolute_path",
			in:   "reading /var/lib/grafana/data.db",
			want: "reading <PATH>",
		},
		{
			name: "hex_blob",
			in:   "hash mismatch deadbeefcafef00d != abc12345",
			want: "hash mismatch <HEX> != <HEX>",
		},
		{
			name: "short_hex_unchanged",
			in:   "code 0xabc",
			want: "code 0xabc",
		},
		{
			name: "long_number",
			in:   "expected 200 got 503 after 15432ms",
			want: "expected <N> got <N> after <N>ms",
		},
		{
			name: "short_number_unchanged",
			in:   "only 2 retries left",
			want: "only 2 retries left",
		},
		{
			name: "combined",
			in:   "2026-03-26T14:57:10Z handler pid=4711 failed for dashboard 550e8400-e29b-41d4-a716-446655440000",
			want: "<TS> handler pid=<PID> failed for dashboard <UUID>",
		},
		{
			name: "truncation",
			in:   longString(maxCanonicalLength + 500),
			want: longString(maxCanonicalLength),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Canonicalize(tc.in)
			if got != tc.want {
				t.Fatalf("Canonicalize(%q):\n  got:  %q\n  want: %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestCanonicalizeStabilityAcrossVariableInputs(t *testing.T) {
	a := Canonicalize("check failed at 2026-03-26T14:57:10.123Z pid=4711 req=abc123def456 took 3421ms")
	b := Canonicalize("check failed at 2026-03-27T09:02:55.001Z pid=9999 req=deadbeefcafef00d took 8123ms")
	if a != b {
		t.Fatalf("expected canonicalised forms to match:\n  a: %q\n  b: %q", a, b)
	}
}

func longString(n int) string {
	// Use 'g' — a non-hex alphabetic — so the hex_blob rule doesn't rewrite
	// the payload while we're testing raw-length truncation.
	b := make([]byte, n)
	for i := range b {
		b[i] = 'g'
	}
	return string(b)
}
