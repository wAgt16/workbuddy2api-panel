package server

import (
	"testing"
	"time"
)

// TestNextMidnightCSTBoundaries 边界：CST 视角下当日 00:00 之后 → 次日 00:00，
// 23:59 → 次日 00:00，正午 → 次日 00:00。用固定 +08:00 偏移计算，不依赖系统时区。
func TestNextMidnightCSTBoundaries(t *testing.T) {
	cst := time.FixedZone("CST", 8*60*60)
	cases := []struct {
		name string
		now  time.Time
		// 期望 until 的 CST 时分秒恒为 00:00:00，且 strictly after now。
	}{
		{"23:59 -> next 00:00", time.Date(2026, 9, 11, 23, 59, 0, 0, cst)},
		{"00:00:01 -> next 00:00", time.Date(2026, 9, 11, 0, 0, 1, 0, cst)},
		{"12:00 -> next 00:00", time.Date(2026, 9, 11, 12, 0, 0, 0, cst)},
		{"00:00:00 exact -> next 00:00", time.Date(2026, 9, 11, 0, 0, 0, 0, cst)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := nextMidnightCST(tc.now)
			if !got.After(tc.now) {
				t.Fatalf("until %v not after now %v", got, tc.now)
			}
			// CST 视角下必须落在 00:00:00。
			if g := got.In(cst); g.Hour() != 0 || g.Minute() != 0 || g.Second() != 0 {
				t.Errorf("until CST = %02d:%02d:%02d, want 00:00:00", g.Hour(), g.Minute(), g.Second())
			}
		})
	}
}

// TestNextMidnightCSTCrossMonth 跨月：月末 23:59 → 次月 1 日 00:00。
func TestNextMidnightCSTCrossMonth(t *testing.T) {
	cst := time.FixedZone("CST", 8*60*60)
	now := time.Date(2026, 1, 31, 23, 59, 0, 0, cst)
	got := nextMidnightCST(now)
	want := time.Date(2026, 2, 1, 0, 0, 0, 0, cst)
	if !got.Equal(want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

// TestDegradeGatePerKeyIsolation 按 system 提示词指纹分桶：一个指纹降级不影响别的
// 指纹（2026-10-25 事故回归：全局布尔量会让一条误报打掉全池所有客户端的 system）。
func TestDegradeGatePerKeyIsolation(t *testing.T) {
	var g degradeGate
	if g.Active("a") || g.Active("b") {
		t.Fatal("fresh gate should not be active")
	}
	g.Trigger("a")
	if !g.Active("a") {
		t.Error("triggered key should be active")
	}
	if g.Active("b") {
		t.Error("other key must stay untouched")
	}
	if g.Active("") {
		t.Error("empty key (no client system) must never be active")
	}
}

// TestDegradeGateTriggerNoRenewal 降级期内再次 Trigger 不续期（保持最早触发点的 00:00 重置）。
func TestDegradeGateTriggerNoRenewal(t *testing.T) {
	var g degradeGate
	g.Trigger("a")
	first := g.untilOf("a")
	g.Trigger("a")
	second := g.untilOf("a")
	if !first.Equal(second) {
		t.Errorf("Trigger during active should not renew: first=%v second=%v", first, second)
	}
}

// TestDegradeGateEmptyKeyIgnored 空指纹（客户端没发 system）不记账：没有 system 可替换，
// 降级无意义。
func TestDegradeGateEmptyKeyIgnored(t *testing.T) {
	var g degradeGate
	g.Trigger("")
	if got := g.untilOf(""); !got.IsZero() {
		t.Errorf("empty key must not be recorded, got %v", got)
	}
}
