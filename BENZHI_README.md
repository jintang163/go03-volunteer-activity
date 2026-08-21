# BENZHI_README

## 项目是做什么的

**go03-volunteer-activity** 是一个志愿者活动管理系统，使用 **纯 Go 标准库**（零第三方依赖）实现。

- **组织者（organizer）** 发布活动（名额、报名窗口、签到码、计划工时），审核报名与候补递补，现场签到/代签，审核工时。
- **志愿者（volunteer）** 浏览广场、报名或进入候补、凭签到码签到签退、查看工时与证书、给已完成活动反馈。
- **管理员（admin）** 管理用户（冻结/解冻）、分类、强制取消活动、全局工时与统计、证书治理。
- **核心业务规则**：满员进候补、录取时校验时间冲突、签到窗口约束、工时按签到签退核算并封顶计划分钟、审核后入账积分、缺席累计限制报名、工时达标颁发证书。
- 内置前端页面（HTML/CSS/JS，`embed` 打包）与文件级 JSON 数据持久化（`data/store.json`，原子落盘，重启自动恢复）。
- 单一 Go 二进制，可通过 Docker 独立运行，适合离线受限环境交付。

技术栈：Go 1.22、`net/http`（Go 1.22 `ServeMux` 方法路由）、`encoding/json`、`embed`、`sync`、`crypto/rand`、`crypto/sha256`。

---

## 构建命令

```bash
# 本地构建（需本地安装 Go 1.22+）
go build ./...

# 质检镜像构建（基于 benzhi.Dockerfile，linux/amd64）
bash ./build_benzhi_docker.sh go03-volunteer-activity
```

## 运行命令

```bash
# 方式一：本地直接运行
go run .

# 方式二：Docker Compose 一键起服务（后台常驻，:8080，种子管理员 admin/admin123）
bash ./go-run.sh
#   等价于：docker compose up -d --build
#   访问：http://localhost:8080/healthz
#   日志：docker compose logs -f
#   停止：docker compose down
```

## 测试命令

```bash
# 方式一：本地测试
go test ./...

# 方式二：质检环境测试（先构建 benzhi 镜像，再在容器内跑 go test）
bash ./go-test.sh go03-volunteer-activity "go test ./..."
```

---

## 目录与质检文件说明

| 文件 | 是否可改 | 说明 |
|------|----------|------|
| `benzhi.Dockerfile` | ❌ 勿改 | 质检镜像（`golang:1.22`，`go mod download` + `go build ./...`） |
| `build_benzhi_docker.sh` | ❌ 勿改 | 质检镜像构建脚本 |
| `go-test.sh` | ✅ 可改 | 质检测试脚本（构建镜像后在容器内执行测试命令） |
| `go-run.sh` | ❌ 勿改 | 运行脚本（`docker compose up -d --build`） |
| `Dockerfile` | ✅ | 运行镜像（单阶段 `golang:1.22`，避免 alpine 拉取超时） |
| `docker-compose.yml` | ✅ | 服务编排（:8080，挂载 `./data` 持久化） |

> 约束：`go.mod` 声明 `go 1.22`，不使用 Go 1.23+ API（如 `crypto/pbkdf2`）；零第三方依赖，确保 `go mod download` 无需联网即可在 `golang:1.22` 镜像内离线构建与测试。

## 默认账号

- 管理员：`admin / admin123`（首次启动自动创建，可通过环境变量 `APP_ADMIN_USERNAME` / `APP_ADMIN_PASSWORD` 覆盖）
- 演示组织者：`organizer / org123`
- 演示志愿者：`alice / alice123`、`bob / bob123`（`APP_SEED_DEMO=true` 时若库中尚无志愿者则写入）

## 快速验证

```bash
# 1. 登录获取 token
curl -s -X POST http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"alice123"}'

# 2. 用返回的 token 报名演示活动（先 GET /api/activities 取 id）
curl -s http://localhost:8080/api/activities \
  -H "Authorization: Bearer <token>"

curl -s -X POST http://localhost:8080/api/activities/<id>/signup \
  -H "Authorization: Bearer <token>" -H 'Content-Type: application/json' \
  -d '{"remark":"可以全程参加"}'

# 3. 浏览器访问 http://localhost:8080/login 查看前端页面
```
