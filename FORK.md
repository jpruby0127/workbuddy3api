# 本分支说明（相对上游 v1.12.0）

> 📘 **项目完整文档请见 [README.md](./README.md)**（功能总览、部署步骤、配置项、API 用法）。
> 本文件只记录**本分支相对上游的改动**、运维注意事项与已知限制。

本分支 `local/expertuse` 基于上游 `v1.12.0-panel`，包含一批本地修复与增强。
上游 `main` 分支已声明停止维护，本分支在此基础上继续运维。

> ⚠️ 本仓库**不包含**任何账号凭证、密钥或运行数据。`auths/`、`data/`、`config.json`、请求日志等均已由 `.gitignore` 排除，首次部署需自行准备。

---

## 目录

- [功能一览](#功能一览)
- [本分支改动](#本分支改动)
- [快速部署](#快速部署)
- [配置说明](#配置说明)
- [任务自动化](#任务自动化)
- [界面说明](#界面说明)
- [安全与数据](#安全与数据)
- [已知限制](#已知限制)

---

## 功能一览

| 模块 | 说明 |
|---|---|
| **API 网关** | `/v1/chat/completions`（非流/流式）、`/v1/models`，OpenAI 协议兼容 |
| **账号池** | 多账号自动选号（加权随机 / 轮询 / 顺序耗尽）、冷却与熔断、积分门槛、到期优先 |
| **签到与保活** | 每日签到、token 保活、余额后台刷新、冷却账号按积分自动解冻 |
| **成长任务** | 全账号任务扫描、执行队列（账号内串行 / 账号间并发 1–4）、自动领奖 |
| **用量** | 请求记录（IP / UA / 模型 / 耗时 / Token / 积分）、成功率、积分构成 |
| **模型档位** | 模型与积分倍率、模型锁池视图 |
| **运行日志** | 按「任务 / 对话 / 系统」分频道、JSONL 归档 |
| **配置** | 面板内热改多数配置，无需重启 |

---

## 本分支改动

相对上游 `v1.12.0-panel`，本分支包含以下提交：

| commit | 改动 |
|---|---|
| `97b73f8` | **`expert_use_*` 类任务自动化**：新增「体验指定专家」动态动作（召唤链 → 真实对话 → `expert_actual_use` 上报 → 自动领奖） |
| `1225daf` | **`WorkBuddy_Valorant` 别名映射**：非 `expert_use_` 前缀的专家任务同样支持 |
| `591d98c` | **`schedule.minute_offset`**：调度时点支持分钟偏移，可设 `00:05` 这类时点 |
| `e357254` | **全站 Neo-Brutalist UI**：零圆角、纯黑边框、硬边阴影、硬切 hover、按压位移；明暗主题统一为白底黑字 |
| `88fdecf` | **任务执行提速**：回读窗口 9s → 2.4s、任务列表本地缓存、专家召唤间隔 6s → 4s、新增「已达标未领自动补领」 |
| `c2c2449` | **手机端响应式**：侧栏改抽屉、表格横向滚动、间距/触控尺寸优化 |

### 1. 指定专家任务自动化

新增文件：

- `internal/panel/autotask_expertuse.go` — 任务码识别、专家名映射、动作实现
- `internal/upstream/expert_search.go` — 专家市场关键词检索

支持的任务码：

| 任务码 | 专家 |
|---|---|
| `expert_use_bowangwang` | 播旺旺（直播电商教练） |
| `expert_use_douyinyin` | 斗音音（抖音策略师） |
| `expert_use_maidehao` | 卖得好（中国电商运营专家） |
| `expert_use_shangjia_xuanchuan` | 商家宣传设计师 |
| `WorkBuddy_Valorant` | 无畏契约拼豆专家 |

新增专家任务时：任务码为 `expert_use_<拼音>` 会自动识别；其他命名在
`internal/panel/autotask_expertuse.go` 的 `expertUseAlias` 里加一行映射即可。

### 2. 调度分钟偏移

```json
{
  "schedule": {
    "minute_offset": 5,
    "checkin_hours": [0, 9, 21],
    "growth_hours": [0]
  }
}
```

`minute_offset` 让所有 `*_hours` 时点按 `XX:offset` 触发（默认 0 = 整点）。
上面这段的效果：**每天 00:05 自动签到并跑一遍全部待办任务**。

### 3. 执行提速

| 项 | 原值 | 现值 |
|---|---|---|
| `claimPollAttempts` | 4 | 3 |
| `claimPollGap` | 3s | 1.2s |
| `expertSummonGap` | 6s | 4s |
| 任务列表查询 | 每个动作一次全量请求 | 阶段 0 拉取后本地索引 |

---

## 快速部署

### 前置

- Docker + Docker Compose
- 一份 `config.json`（从 `config.example.json` 复制后改）
- 一份 `auths/` 目录（在面板「添加账号」里登录后自动生成）

### 步骤

```bash
git clone <本仓库地址> workbuddy2api-panel
cd workbuddy2api-panel

cp config.example.json config.json   # 按需修改：api_key / listen / 调度时点等

# 构建镜像（国内环境需要 Go 模块代理，Dockerfile 已内置 ENV GOPROXY）
docker build -t workbuddy2api:local .

# 按实际端口/数据卷调整 docker-compose.yml 后启动
docker compose up -d

# 健康检查
docker ps --filter name=workbuddy2api
curl -I http://127.0.0.1:7863/panel/
```

面板地址：`http://<服务器IP>:7863/panel/`

> 国内服务器构建时若 `go mod download` 超时，请保留 `Dockerfile` 里的
> `ENV GOPROXY=https://mirrors.tencent.com/go,direct`（本分支已保留该行）。

---

## 配置说明

配置文件为 `config.json`，面板「配置」页可热改多数项。以下是运维中常动的几项：

```json
{
  "api_key": "用于 /v1 与面板 API 的 Bearer 密钥",
  "schedule": {
    "checkin_enabled": true,
    "checkin_hours": [0, 9, 21],
    "growth_enabled": true,
    "growth_hours": [0],
    "minute_offset": 5,
    "balance_refresh_minutes": 5
  },
  "pool": {
    "strategy": { "domestic": "weighted_random", "international": "weighted_random" },
    "max_in_flight": 3,
    "credit_floor": 100
  }
}
```

- **`schedule.minute_offset`**（本分支新增）：0–59，越界回落 0；>0 时各时点按 `XX:offset` 触发
- **`pool.strategy`**：`weighted_random` / `round_robin` / `sequential_exhaust`
- 新增字段均有默认兜底，旧 `config.json` 缺字段不会导致故障

---

## 任务自动化

两个入口都会执行「有实现」的任务：

- 面板 → 账号 → **一键完成全部**（`POST /panel/api/accounts/{uid}/tasks/auto_all`）
- 任务中心 → **执行全部待办**（`POST /panel/api/tasks/run_queue`，带 `concurrency` 1–4）

### 已实现的任务码

```
chat_5  first_buddy  Model_chat_GLM5.2  RichMeow_Chat  Buddy_App  Buddy_App_QQ
automation_1  Library_read  template_5  playbook_prompt  create_canvas
expert_5  Expert_team_use_3  Hp_Appearance  skill_1  Expert_lighthouse
black_cat  school_season  Sequential_Tasks_1..7
expert_use_*（动态前缀）  WorkBuddy_Valorant（别名）
```

### 不会执行的任务

- **`wb_wechat_oa_subscribe_task`（关注公众号）**：需要真实微信关注动作并绑定微信 openid，纯 API 无法完成，只能手动扫码
- **上游 `locked` 的任务**：如 Sequential 系列未开放的一环，等上游下发目标后会自动进入待办

### 关于「进度 not_accepted」

上游计分是异步的，队列结果里出现的「进度 not_accepted」是执行当时的回读快照，
通常几秒后即入账。以任务列表里的最终状态为准。

---

## 界面说明

### Neo-Brutalist 视觉（本分支）

- 零圆角、纯黑边框（移动端 2px / 桌面 4px）
- 硬边阴影 `4px 4px 0 #000`（桌面 `8px 8px 0 #000`），无模糊
- hover 硬切高对比色（粉 `#ff006e` / 蓝 `#00d9ff`），`150ms ease-out`
- 按钮按下位移 = 阴影像素值并移除阴影（实体压平）
- 强调色：粉 `#ff006e`（CTA/错误）、绿 `#ccff00`（选中/成功）、蓝 `#00d9ff`（信息/hover）、橙 `#ff9500`（警示）

样式集中在 `internal/panel/index.html` 末尾的 **覆盖层**，不改动 HTML 结构与业务逻辑，便于回退。

### 手机端（≤760px）

- 侧栏 → 抽屉（左上角 ☰ 打开，点遮罩 / ESC / 点导航项关闭）
- 表格横向滚动、统计卡两列、筛选条换行、按钮触控高度 ≥36px
- 窗口转回宽屏时抽屉自动复位

---

## 安全与数据

以下数据**不入库**，部署时需自行准备：

| 路径 | 内容 |
|---|---|
| `auths/` | 账号凭证（access token / refresh token） |
| `data/` | 账号池状态、用量桶 |
| `config.json` | 含 `api_key` 等密钥（模板见 `config.example.json`） |
| `request-logs/` | 请求日志归档 |

`.gitignore` 已排除上述路径。推送前请确认：

```bash
git status --short
git ls-files | grep -E 'auths/|config.json|^data/'   # 应无输出
```

---

## 已知限制

1. **关注公众号类任务**无法自动化（需真实微信操作）
2. **企业切换会话**的账号（JWT `token_source=enterprise_switch`）部分行为事件会被上游静默丢弃，表现为任务反复执行但进度不涨
3. Sequential 系列每天只解锁一环，未开放的环无法提前执行
4. 上游主分支已声明停止维护，本分支为本地维护，不保证与上游后续变更同步

---

## 目录结构

```
cmd/server/          服务入口与配置
internal/auth/       账号凭证与 token 刷新
internal/pool/       选号策略、冷却、积分与到期
internal/scheduler/  定时任务（签到/旅行/活跃/保活/夜猫子/成长队列）
internal/server/     OpenAI 兼容网关
internal/upstream/   上游协议（对话/任务/余额/专家/开学季）
internal/panel/      控制台（index.html + app.js）
```

---

## 许可证

沿用上游项目许可证。使用本工具请遵守上游平台的服务条款，账号与配额风险由使用者自行承担。
