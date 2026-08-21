# 志愿者活动管理系统（go03-volunteer-activity）

一个使用 **纯 Go 标准库** 从 0 到 1 构建的志愿者活动管理系统。组织者发布活动、志愿者报名、现场签到签退、系统按规则核算工时并沉淀积分与证书。系统内置前端页面与文件级 JSON 数据持久化，可通过 Docker 独立运行。

---

## 一、项目简介

社区、学校、公益组织常见的痛点是：活动散落在表格和群聊里，名额对不上、到场核销靠点名、工时事后补登且无法追溯。本系统把四条主链路产品化：

- **发布活动**：组织者创建草稿、配置名额/报名截止/签到窗口，发布后广场可见。
- **报名**：志愿者报名；满员进入候补；组织者可审核、从候补递补、拒绝或取消。
- **签到**：活动进行中，志愿者凭签到码自助签到/签退，组织者可代签。
- **工时记录**：根据签到签退自动核算分钟数，经组织者/管理员审核后入账，驱动积分与证书。

角色：

- **志愿者（volunteer）**：注册登录、浏览活动、报名/候补、签到签退、查看个人工时与证书、给活动反馈。
- **组织者（organizer）**：发布与管理自己的活动、审核报名、现场签到管理、审核工时、组建队伍。
- **管理员（admin）**：用户与分类治理、强制取消活动、全局统计、发放证书、处理异常工时。

系统使用 Go 1.22 + 标准库（`net/http`、`encoding/json`、`embed`、`sync` 等），**零第三方依赖**，可完全离线构建与运行。

---

## 二、功能特性

### 2.1 用户与权限

| 角色 | 能力 |
|------|------|
| 管理员 | 用户列表/创建/冻结/解冻，分类管理，强制取消活动，审核任意工时，查看全局统计，颁发证书 |
| 组织者 | 发布活动、管理报名与候补、生成签到码、代签到、提交/审核工时、管理自己的队伍 |
| 志愿者 | 注册登录、报名、签到签退、查看工时流水、申请工时更正、提交活动反馈 |
| 未登录访客 | 仅登录/注册页与健康检查 |

- 首次启动自动创建种子管理员（默认 `admin / admin123`）。
- 志愿者可自助注册；组织者账号由管理员创建（避免随意发活动）。
- 会话：Bearer Token，带过期时间；登出、改密、冻结即失效。
- 口令：盐值 + 多轮迭代 SHA-256（演示级，生产应替换为 bcrypt/argon2）。
- 账号状态：`active` / `frozen`（冻结后不可报名、发活动） / `banned`（无法登录）。

### 2.2 活动生命周期（发布）

状态流转：

```
[ 草稿 draft ]
      │ 发布 publish
      ▼
[ 已发布 published ] ──报名截止或组织者关闭报名──► [ 报名截止 registration_closed ]
      │                                                      │
      │ 到达活动开始时间（惰性推进）                         │ 组织者开场
      ▼                                                      ▼
[ 进行中 in_progress ] ◄────────────────────────────────────┘
      │
      │ 组织者结项 / 超过结束时间惰性结项
      ▼
[ 已完成 completed ]

任意非终态可由组织者或管理员 ──取消──► [ 已取消 cancelled ]
```

活动字段要点：

- **分类**：环保、助老、支教、社区服务、应急、文化、体育、其他。
- **名额 `Capacity`**：已录取人数达到名额后，新报名进入候补（若开启候补）或直接拒绝。
- **候补 `WaitlistEnabled` / `WaitlistLimit`**。
- **报名窗口**：`SignupOpenAt` ~ `SignupCloseAt`；未到开放时间不可报。
- **活动时间**：`StartAt` ~ `EndAt`。
- **签到窗口**：开始前 `CheckInOpenBefore` 分钟可签到，结束后 `CheckOutGrace` 分钟内可签退。
- **是否需要审核 `NeedApproval`**：关闭时报名即录取（未满员）；开启时先进入 pending。
- **签到码 `CheckInCode`**：发布时生成，组织者可刷新。
- **所需技能、最低年龄、集合地点、联系人、计划工时上限 `PlannedMinutes`**。
- **队伍（可选）**：活动可绑定一个队伍，仅队员可报名（用于固定志愿队）。

### 2.3 报名规则（核心）

> **定义**：同一志愿者对同一活动最多一条有效报名（pending / approved / waitlisted）。撤回或拒绝后可再次报名（若窗口仍开放）。

1. 不能给自己组织的活动报名。
2. 活动必须为 `published` 或 `registration_closed` 后不允许新报名（截止后只处理已有名单）。
3. **满员**：已 `approved` 数量 ≥ Capacity：
   - 开启候补且候补未满 → 状态 `waitlisted`，记录候补序号。
   - 否则拒绝并返回冲突错误。
4. **NeedApproval=true**：未满员时进入 `pending`，组织者通过后变 `approved`。
5. **NeedApproval=false**：未满员直接 `approved`。
6. **时间冲突**：同一志愿者已录取且时间窗重叠的活动不可再被录取（候补允许，递补时再检查）。
7. **信誉门槛**：近 90 天无故缺席（no-show）≥ 3 次，禁止新报名，需管理员解禁。
8. **取消报名**：录取后、活动开始前可取消；开始后取消记一次违约（影响信誉）。
9. **候补递补**：有人取消或组织者拒绝已录取者后，按候补序号 FIFO 自动尝试递补；时间冲突则跳过该候补并继续下一名。
10. 组织者可批量录取 pending、拒绝并填写原因。

报名状态：`pending` / `approved` / `waitlisted` / `rejected` / `cancelled` / `no_show`。

### 2.4 签到规则（核心）

> **定义**：仅 **已录取（approved）** 志愿者可签到。一次活动每人最多一条签到记录（可多次修正签退时间，由组织者操作）。

1. **自助签到**：提交活动签到码；必须落在签到窗口内。
2. **组织者代签**：无需码，可补录 `CheckInAt`（不得早于窗口起点、不得晚于当前时间过多，受策略约束）。
3. **签退**：已签到且未签退；窗口为活动结束 + `CheckOutGrace`。
4. 未签到而活动已结束超过宽限：系统在结项时将报名标为 `no_show`。
5. 重复签到返回冲突。
6. 签到方式记入 `Method`：`self` / `organizer` / `code`。

签到记录字段：`CheckInAt`、`CheckOutAt`、`Method`、`Note`、`ActorID`。

### 2.5 工时记录（核心）

> **定义**：工时以 **分钟** 入账。自动工时来自签到签退；手工工时由组织者申报。只有 **已审核（approved）** 的工时计入志愿者累计时长与积分。

核算公式：

```
rawMinutes = CheckOutAt - CheckInAt
workMinutes = min(max(rawMinutes - BreakMinutes, 0), PlannedMinutes)
```

- `BreakMinutes` 由组织者在审核时填写（休息/集合空档），默认 0。
- 未签退不得自动生成工时；可改由组织者提交手工工时（`Source=manual`）。
- 同一（活动, 志愿者）同时只能有一条 **待审或已通过** 工时；被驳回后可重新提交。
- 志愿者可对已通过工时发起 **更正申请**（说明实际分钟），组织者/管理员再审。
- 活动未完成前允许预生成待审工时，但结项时会按最终签退重算（若尚未审核）。
- 审核通过后：
  - 累加 `User.TotalMinutes`
  - 积分 `Points += workMinutes / 6`（每 6 分钟 1 分，即每小时 10 分）
  - 写工时流水与积分流水
- 管理员可冲正（负数流水），积分不低于 0。

工时状态：`draft`（自动生成待确认） / `pending` / `approved` / `rejected` / `corrected`。

### 2.6 积分、证书、队伍

- **积分**：初始 0。完成工时获得；无故缺席 -15；开始后取消报名 -8。
- **证书**：累计有效工时达到档位自动或由管理员颁发：
  - 铜章 20 小时、银章 50 小时、金章 100 小时。
  - 证书编号可验证（`GET /api/certificates/{code}/verify` 公开接口）。
- **队伍**：组织者创建队伍、邀请志愿者、队员可看到队内活动；绑定队伍的活动仅队员可报名。

### 2.7 反馈、通知、审计

- **反馈**：活动 completed 后，已签到志愿者可评分 1–5 并留言；每活动每人一次。
- **通知**：报名结果、递补成功、签到提醒（读取时按开始时间惰性生成逻辑由列表提示承担）、工时审核结果、证书颁发。
- **审计日志**：发布、录取、签到、工时审核、取消活动等写 `AuditLog`，管理员可检索。

### 2.8 统计

管理员看板：用户数、活动数（按状态）、报名转化、平均满员率、本月工时合计、缺席率。  
组织者看板：自己活动的录取人数、到场人数、待审工时。  
志愿者主页：累计工时、积分、证书、即将开始的已报名活动。

---

## 三、业务对象与持久化

数据全部保存在 `data/store.json`（路径由 `APP_DATA_PATH` 配置）。内存结构变更后通过钩子 **原子写盘**（临时文件 + rename），进程重启后恢复。

主要集合：

- Users、Activities、Signups、CheckIns、HourRecords、HourLedgers
- Teams、TeamMembers、Notifications、AuditLogs、Certificates、Feedbacks
- Categories、PointLedgers

并发：`MemoryStore` 使用 `sync.RWMutex`；跨实体操作（录取+递补、工时入账+积分）在同一把锁内完成，避免半更新。

---

## 四、API 一览

前缀 `/api`，除登录注册、健康检查、证书核验外均需 `Authorization: Bearer <token>`。

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/healthz` | 健康检查 |
| POST | `/api/auth/register` | 志愿者注册 |
| POST | `/api/auth/login` | 登录 |
| POST | `/api/auth/logout` | 登出 |
| GET | `/api/auth/me` | 当前用户 |
| PUT | `/api/me/profile` | 改资料 |
| PUT | `/api/me/password` | 改密 |
| GET | `/api/categories` | 分类 |
| GET/POST | `/api/activities` | 列表 / 创建草稿 |
| GET/PUT | `/api/activities/{id}` | 详情 / 更新草稿 |
| POST | `/api/activities/{id}/publish` | 发布 |
| POST | `/api/activities/{id}/close-signup` | 关闭报名 |
| POST | `/api/activities/{id}/start` | 开场 |
| POST | `/api/activities/{id}/complete` | 结项 |
| POST | `/api/activities/{id}/cancel` | 取消 |
| POST | `/api/activities/{id}/refresh-code` | 刷新签到码 |
| POST | `/api/activities/{id}/signup` | 报名 |
| GET | `/api/activities/{id}/signups` | 报名名单 |
| POST | `/api/signups/{id}/approve` | 录取 |
| POST | `/api/signups/{id}/reject` | 拒绝 |
| POST | `/api/signups/{id}/cancel` | 取消报名 |
| POST | `/api/activities/{id}/checkin` | 签到 |
| POST | `/api/activities/{id}/checkout` | 签退 |
| POST | `/api/activities/{id}/proxy-checkin` | 代签到 |
| GET | `/api/activities/{id}/checkins` | 签到列表 |
| POST | `/api/hours` | 提交手工工时 |
| GET | `/api/me/hours` | 我的工时 |
| POST | `/api/hours/{id}/approve` | 审核通过 |
| POST | `/api/hours/{id}/reject` | 驳回 |
| POST | `/api/hours/{id}/correct` | 更正申请 |
| GET | `/api/me/notifications` | 通知 |
| GET/POST | `/api/teams` | 队伍 |
| POST | `/api/teams/{id}/members` | 邀请队员 |
| GET | `/api/stats` | 统计（管理员） |
| GET | `/api/certificates/{code}/verify` | 证书核验（公开） |

前端页面（`embed`）：`/login`、`/app`（广场与报名）、`/activity/{id}`、`/me`、`/organizer`、`/admin`。

---

## 五、技术架构

```
main.go
  ├─ config          环境变量
  ├─ store           MemoryStore + FileStore 原子持久化
  ├─ auth            口令哈希 / 会话
  ├─ service         活动 / 报名 / 签到 / 工时 / 积分证书 / 通知
  ├─ handler         HTTP JSON + 页面
  ├─ middleware      鉴权、角色、CORS、日志、Recover
  └─ web/assets      内置 HTML/CSS/JS
```

约束：

- `go.mod` 声明 `go 1.22`，不使用 Go 1.23+ API。
- 零第三方模块，质检镜像 `golang:1.22` 内 `go mod download` 与 `go build ./...` 可离线完成。
- 运行镜像见项目根 `Dockerfile` + `docker-compose.yml`（端口 8080，挂载 `./data`）。

---

## 六、默认账号与演示数据

| 账号 | 口令 | 角色 |
|------|------|------|
| admin | admin123 | 管理员 |
| organizer | org123 | 组织者 |
| alice | alice123 | 志愿者 |
| bob | bob123 | 志愿者 |

`APP_SEED_DEMO=true` 时写入上述账号及一条可报名的演示活动（社区环保清洁）。

环境变量：`APP_ADDR`、`APP_DATA_PATH`、`APP_SESSION_TTL`、`APP_ADMIN_USERNAME`、`APP_ADMIN_PASSWORD`、`APP_SEED_ADMIN`、`APP_SEED_DEMO`。

---

## 七、本地与 Docker 运行

见 `BENZHI_README.md`。简述：

```bash
go run .
# 或
bash ./go-run.sh
```

浏览器打开 http://localhost:8080/login 。

---

## 八、核心规则速查

| 场景 | 结果 |
|------|------|
| 满员且开启候补 | 报名状态 waitlisted |
| 满员且无候补 | 409 冲突 |
| 时间重叠的另一场已录取 | 不可再录取（候补可排队） |
| 90 天缺席 ≥ 3 | 禁止报名 |
| 无签到且活动结束超宽限 | 报名标 no_show，扣积分 |
| 有签到无签退 | 不能自动出工时，需手工申报 |
| 工时超过计划分钟 | 按 PlannedMinutes 封顶 |
| 累计工时跨过 20/50/100 小时 | 颁发对应证书 |
