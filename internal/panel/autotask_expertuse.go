package panel

// expert_use_<专家拼音> 类任务（如 expert_use_bowangwang「体验直播电商教练」）的动态动作。
//
// 这类任务码随平台专家库新增而不断增加，autoActions 固定表无法穷举，故按
// "expert_use_" 前缀动态识别：解析后缀 → 定位专家市场里的真实专家 → 召唤链 →
// 真实对话（服务端 requestId）→ expert_actual_use 上报。
//
// 另有一批"体验指定专家"任务不走 expert_use_ 前缀（如 WorkBuddy_Valorant
// 「无畏契约拼豆专家」），用 expertUseAlias 按任务码直接映射到专家显示名。
//
// 判据与 Expert_lighthouse 同构：真实专家 id + 真实 chat requestId，两者缺一不计数。
// 实测对应关系（2026-10-24，市场接口 keyword 检索核对）：
//   bowangwang         → 播旺旺          livestream-ecommerce-coach
//   douyinyin          → 斗音音          douyin-strategist
//   maidehao           → 卖得好          china-ecommerce-operations-expert
//   shangjia_xuanchuan → 商家宣传设计师   promo-page-designer
//   WorkBuddy_Valorant → 无畏契约拼豆专家 valorant-perler-expert
//
// 任务码后缀取专家显示名的拼音；新任务在此登记即可精确命中，未登记的走任务标题
// 「」内关键词回退（取搜索首条真实专家）。

import (
	"fmt"
	"strings"

	"github.com/linguo2625469/workbuddy2api-panel/internal/auth"
	"github.com/linguo2625469/workbuddy2api-panel/internal/upstream"
)

const expertUsePrefix = "expert_use_"

// expertUseNames 任务码后缀（专家显示名全拼）→ 专家显示名。
var expertUseNames = map[string]string{
	"bowangwang":         "播旺旺",
	"douyinyin":          "斗音音",
	"maidehao":           "卖得好",
	"shangjia_xuanchuan": "商家宣传设计师",
}

// expertUseAlias 不走 expert_use_ 前缀、但同样要求"体验指定专家"的任务码 → 专家显示名。
var expertUseAlias = map[string]string{
	"WorkBuddy_Valorant": "无畏契约拼豆专家",
}

// isExpertUseCode 报告任务码是否属于指定专家体验类任务。
func isExpertUseCode(code string) bool {
	c := strings.TrimSpace(code)
	if strings.HasPrefix(c, expertUsePrefix) {
		return true
	}
	_, ok := expertUseAlias[c]
	return ok
}

// expertUseSuffix 去掉前缀后的专家拼音。
func expertUseSuffix(code string) string {
	return strings.TrimPrefix(strings.TrimSpace(code), expertUsePrefix)
}

// expertUseDisplayName 任务码要求的专家显示名（别名优先，其次拼音表，无则空）。
func expertUseDisplayName(code string) string {
	c := strings.TrimSpace(code)
	if n, ok := expertUseAlias[c]; ok {
		return n
	}
	return expertUseNames[expertUseSuffix(c)]
}

// expertUseDesc 任务码的展示说明。
func expertUseDesc(code string) string {
	if n := expertUseDisplayName(code); n != "" {
		return "体验指定专家「" + n + "」（召唤 + 真实对话 + 使用上报）"
	}
	return "体验指定专家（召唤 + 真实对话 + 使用上报）"
}

// runExpertUseByCode 完成指定专家体验类任务。
// 链路与 runExpertLighthouse 同构（该形态已两账号点亮）：召唤链 → 真实对话拿
// 服务端 requestId → chat 链注入 has_expert/expert_id → expert_actual_use。
func runExpertUseByCode(p *Panel, a *auth.Auth, code string) (string, error) {
	e, err := findExpertForUseCode(p, a, code)
	if err != nil {
		return "", err
	}
	if err := p.cfg.Upstream.ReportDesktopEvent(a, upstream.DesktopExpertSummonSequence(e)...); err != nil {
		return "", fmt.Errorf("召唤链: %w", err)
	}
	conv, req, err := p.cfg.Upstream.DesktopChatWithExpert(a, e.ExpertID)
	if err != nil {
		return "", fmt.Errorf("真实对话: %w", err)
	}
	events := upstream.DesktopChatSequence(conv, req, "msg-"+req[len(req)-8:], "fast-model", "fast-model")
	for _, ev := range events {
		if ev["eventCode"] == "agent_task_created" {
			ev["has_expert"] = true
			ev["expert_id"] = e.ExpertID
			ev["expert_name"] = e.DisplayNameZH
			ev["expert_industry_id"] = ""
		}
	}
	events = append(events, upstream.DesktopExpertActualUseLocal(e, conv, req))
	events[len(events)-1]["type"] = ""
	events[len(events)-1]["cost"] = 0
	if err := p.cfg.Upstream.ReportDesktopEvent(a, events...); err != nil {
		return "", fmt.Errorf("使用事件: %w", err)
	}
	return fmt.Sprintf("已上报「%s」召唤+使用链（真实对话 requestId）", e.DisplayNameZH), nil
}

// findExpertForUseCode 定位任务要求的真实专家。
// 先按登记的显示名精确搜索；未登记的任务码用任务标题「」内关键词兜底。
func findExpertForUseCode(p *Panel, a *auth.Auth, code string) (upstream.MarketExpert, error) {
	var zero upstream.MarketExpert
	name := expertUseDisplayName(code)
	var keywords []string
	if name != "" {
		keywords = append(keywords, name)
	}
	if t, err := p.taskByCode(a, code); err == nil && t != nil {
		if kw := taskTitleKeyword(t.Title); kw != "" && kw != name {
			keywords = append(keywords, kw)
		}
	}
	if len(keywords) == 0 {
		return zero, fmt.Errorf("任务 %s 无法确定专家关键词", code)
	}
	for _, kw := range keywords {
		list, err := p.cfg.Upstream.MarketExpertSearch(a, kw)
		if err != nil || len(list) == 0 {
			continue
		}
		for _, e := range list {
			if name != "" && e.DisplayNameZH == name {
				return e, nil
			}
		}
		if name == "" {
			return list[0], nil
		}
	}
	return zero, fmt.Errorf("专家市场未找到任务 %s 对应的真实专家（关键词: %s）", code, strings.Join(keywords, "/"))
}

// taskTitleKeyword 取任务标题中「」/『』包裹的内容（"体验「直播电商教练」" → "直播电商教练"）。
func taskTitleKeyword(title string) string {
	for _, pair := range [][2]string{{"「", "」"}, {"『", "』"}} {
		if i := strings.Index(title, pair[0]); i >= 0 {
			rest := title[i+len(pair[0]):]
			if j := strings.Index(rest, pair[1]); j > 0 {
				return strings.TrimSpace(rest[:j])
			}
		}
	}
	return ""
}
