package upstream

import (
	"net/http"
	"testing"
)

// TestClassifyChannelRefusedPhrasings 11-128 通道校验失败的两种文案都必须归
// ErrChannelRefused（不罚号、**不触发提示词降级**），而不是 ErrContentBlocked。
//
// 回归背景（2026-10-25 实机事故）：上游文案变成 "Illegal API invocation from an
// unapproved channel"，旧实现只认 "request illegal"，于是落进 contentBlockedMarkers
// → ErrContentBlocked → handler 把全池客户端的 system 提示词换成中性文本直到次日
// 00:00（表现为 dsh 的 agent 指令与会话标题格式要求全部失效）。
func TestClassifyChannelRefusedPhrasings(t *testing.T) {
	bodies := []string{
		`{"code":11-128,"msg":"Illegal API invocation from an unapproved channel"}`,
		`{"code":"11-128","msg":"request illegal"}`,
		`{"code":11-128,"msg":"blocked by an unapproved channel"}`,
	}
	for _, b := range bodies {
		if got := Classify(http.StatusBadRequest, b); got != ErrChannelRefused {
			t.Errorf("Classify(%s) = %v, want ErrChannelRefused", b, got)
		}
	}
}

// TestClassifyContentBlockedStillContent 真正的内容审核文案仍归 ErrContentBlocked
// （降级重试的唯一触发条件）。
func TestClassifyContentBlockedStillContent(t *testing.T) {
	b := `{"code":11-128,"msg":"blocked by security policy"}`
	if got := Classify(http.StatusBadRequest, b); got != ErrContentBlocked {
		t.Errorf("Classify(content block) = %v, want ErrContentBlocked", got)
	}
}
