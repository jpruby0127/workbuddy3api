package upstream

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/linguo2625469/workbuddy2api-panel/internal/auth"
)

// MarketExpertSearch 按关键词搜索专家市场。
//
// 与 MarketExpertList 同端点，仅多一个 keyword 字段（上游在该字段做全文匹配，
// 实测「直播电商教练」命中 6 条、「商家宣传页与海报设计师」命中 1 条）。
// 供面板 expert_use_<专家拼音> 类任务定位任务要求的那个真实专家：上游判据
// 校验 expert_actual_use 的 id 必须是平台上真实存在的专家（编造 id 不计数）。
func (c *Client) MarketExpertSearch(a *auth.Auth, keyword string) ([]MarketExpert, error) {
	body := map[string]any{"page": 1, "page_size": 10, "expert_type": "agent", "keyword": keyword}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, c.chatBase(a)+"/portal/operation-platform/market/expert/list", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+a.AccessTokenValue())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", desktopUA)
	req.Header.Set("X-Domain", c.chatBase(a))
	req.Header.Set("X-Product", "SaaS")
	if a.UID != "" {
		req.Header.Set("X-User-Id", a.UID)
	}
	var out struct {
		Experts []MarketExpert `json:"experts"`
	}
	data, err := c.doJSON(req)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("expert search parse: %w", err)
	}
	return out.Experts, nil
}
