package prompt

import "testing"

// TestSystemFingerprint 指纹只取开头连续 system/developer 块，且对同一提示词稳定。
func TestSystemFingerprint(t *testing.T) {
	a := []byte(`{"messages":[{"role":"system","content":"A"},{"role":"user","content":"hi"}]}`)
	b := []byte(`{"messages":[{"role":"system","content":"A"},{"role":"user","content":"different"}]}`)
	c := []byte(`{"messages":[{"role":"system","content":"B"},{"role":"user","content":"hi"}]}`)
	d := []byte(`{"messages":[{"role":"user","content":"hi"}]}`)
	if SystemFingerprint(a) == "" || SystemFingerprint(a) == "none" {
		t.Fatalf("expected a real fingerprint, got %q", SystemFingerprint(a))
	}
	if SystemFingerprint(a) != SystemFingerprint(b) {
		t.Error("same system prompt must yield the same fingerprint regardless of user text")
	}
	if SystemFingerprint(a) == SystemFingerprint(c) {
		t.Error("different system prompts must yield different fingerprints")
	}
	if got := SystemFingerprint(d); got != "none" {
		t.Errorf("no system message => %q, want \"none\"", got)
	}
	if got := SystemFingerprint([]byte("not json")); got != "unparsable" {
		t.Errorf("bad body => %q, want \"unparsable\"", got)
	}
}
