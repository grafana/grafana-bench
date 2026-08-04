package analyzer

import "testing"

func TestSignatureDeterministic(t *testing.T) {
	a := Signature("permissions.ts", "check failed: folder.get expected 200 got 403")
	b := Signature("permissions.ts", "check failed: folder.get expected 200 got 403")
	if a != b {
		t.Fatalf("expected identical inputs to produce identical signatures, got %q vs %q", a, b)
	}
	if len(a) != signatureLength {
		t.Fatalf("expected signature length %d, got %d (%q)", signatureLength, len(a), a)
	}
}

func TestSignatureDifferentiatesByTestFile(t *testing.T) {
	msg := "check failed: folder.get expected 200 got 403"
	a := Signature("permissions.ts", msg)
	b := Signature("folder_list.ts", msg)
	if a == b {
		t.Fatalf("expected different test files to produce different signatures, got %q for both", a)
	}
}

func TestSignatureDifferentiatesByMessage(t *testing.T) {
	file := "permissions.ts"
	a := Signature(file, "check failed: folder.get expected <N> got <N>")
	b := Signature(file, "check failed: dashboard.get expected <N> got <N>")
	if a == b {
		t.Fatalf("expected different messages to produce different signatures, got %q for both", a)
	}
}
