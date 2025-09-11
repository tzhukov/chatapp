package crypto

import "testing"

func TestHashAndCompare(t *testing.T) {
	h, err := Hash("s3cret")
	if err != nil {
		t.Fatalf("hash error: %v", err)
	}
	if !Compare(h, "s3cret") {
		t.Fatalf("expected match")
	}
	if Compare(h, "wrong") {
		t.Fatalf("expected mismatch")
	}
}
