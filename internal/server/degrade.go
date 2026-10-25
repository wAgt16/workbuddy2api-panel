package server

import (
	"sync"
	"time"
)

// degradeGate 降级状态机：passthrough 模式下请求被上游内容策略拦截、且换成中性
// 提示词**重试成功**时，记住「这条 system 提示词会撞审核」，之后携带同一提示词的
// 请求直达 Degraded，不再先撞 400。按 system 提示词指纹分桶，直到次日 00:00 CST
// 重置。
//
// 定位：误报处理（与 internal/upstream/sanitize.go 同哲学）——内容拦截多为
// system 来源的指纹误报，换最小中性提示词即可绕开；不是对抗框架。
//
// **为什么按指纹分桶，而不是全局一个布尔量**（2026-10-25 实机事故）：旧实现一撞
// 就把整个网关切成「替换所有客户端 system」，直到次日 00:00。一个客户端的误报
// 于是打掉全池所有客户端的 system 提示词 —— dsh 的 agent 指令、会话标题插件的
// 格式要求全部消失（表现：标题只剩「日期｜主题」，`{type}` 永远为空），而且
// 从日志上完全看不出原因。分桶后只有真正撞墙的那条提示词被替换。
//
// **为什么只在重试成功后记账**：内容拦截也可能是**用户正文**触发的。此时换中性
// 提示词重试同样失败，说明与 system 无关，不该留下任何状态（旧实现在重试前就
// 无条件记账，一次用户正文拦截能污染整个进程生命周期 8 小时）。
//
// 状态机：进程内存、重启清零（可接受：重启极少触发，且 custom 模式根本
// 不进降级路径）。
type degradeGate struct {
	mu     sync.Mutex
	active map[string]time.Time // system 提示词指纹 → 降级截止时刻
}

// Active 报告某个提示词指纹当前是否处于降级期（now < until）。
// key 为空串（客户端没发 system）时返回 false：没有 system 可替换，降级无意义。
func (g *degradeGate) Active(key string) bool {
	if key == "" {
		return false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	until, ok := g.active[key]
	if !ok {
		return false
	}
	return time.Now().Before(until)
}

// Trigger 让某个提示词指纹进入降级期，直到次日 00:00 CST（Asia/Shanghai）。
// 已在降级期内则不续期（保持最早触发点的 00:00 重置语义）。
// 空 key 直接忽略（见 Active）。
func (g *degradeGate) Trigger(key string) {
	if key == "" {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.active == nil {
		g.active = make(map[string]time.Time)
	}
	if until, ok := g.active[key]; ok && time.Now().Before(until) {
		return // 降级期内不续期
	}
	g.active[key] = nextMidnightCST(time.Now())
}

// untilOf 返回某指纹的降级截止时刻（测试与排障用；不存在返回零值）。
func (g *degradeGate) untilOf(key string) time.Time {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.active[key]
}

// nextMidnightCST 返回 now 之后最近的 Asia/Shanghai 00:00 时刻（纯函数，方便单测）。
//
// 边界语义：
//   - 23:59 → 次日 00:00（几秒后）
//   - 00:00 → 次日 00:00（刚过 00:00，下个零点是次日）
//
// 用固定 +08:00 偏移计算，避免依赖系统时区配置（容器/宿主机时区不确定）。
func nextMidnightCST(now time.Time) time.Time {
	cst := time.FixedZone("CST", 8*60*60)
	// 把 now 转到 CST 视角取当天 00:00，再加一天；若该 00:00 <= now 则再加一天。
	y, m, d := now.In(cst).Date()
	midnight := time.Date(y, m, d, 0, 0, 0, 0, cst)
	for !midnight.After(now) {
		midnight = midnight.Add(24 * time.Hour)
	}
	return midnight
}
