package upstream

import "testing"

// TestSanitizeDeepSeekHarnessBrand CN 域（copilot.tencent.com）的渠道校验按 system
// 提示词里的 "DeepSeek Harness" 拦整单（400 code=11-128 "Illegal API invocation from
// an unapproved channel"，2026-10-25 实测）。净化只改一个词，语义不变，但足以让整份
// agent 提示词通过渠道校验——不换就只能靠降级把 system 整段换成中性文本。
func TestSanitizeDeepSeekHarnessBrand(t *testing.T) {
	in := "You are an AI agent powered by DeepSeek Harness."
	want := "You are an AI agent powered by DeepSeek."
	if got := sanitizeText(in); got != want {
		t.Errorf("sanitizeText(%q) = %q, want %q", in, got, want)
	}
	in2 := "DeepSeek Harness checkout. DeepSeek Harness Web GUI."
	want2 := "DeepSeek checkout. DeepSeek Web GUI."
	if got := sanitizeText(in2); got != want2 {
		t.Errorf("sanitizeText(%q) = %q, want %q", in2, got, want2)
	}
}
