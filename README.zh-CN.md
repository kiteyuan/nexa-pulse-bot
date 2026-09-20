# NexaPulse

[English](./README.md) | **简体中文**

自托管资讯服务：Telegram / RSS 采集，规则过滤，可选 LLM 整理，公开资讯站浏览。

- **Server**：Go。采集、处理、存储。
- **PostgreSQL**：账号、频道、消息、栏目、设置。
- **Admin**：React + TypeScript。宿主机只绑 `127.0.0.1:8081`（不要改成 `0.0.0.0:8081`）。
- **Public**：React + TypeScript，默认 `:8080`。

```text
Telegram / RSS ──► Go server ──► PostgreSQL
                      │
              ┌───────┴────────┐
              ▼                ▼
            Admin            Public
```

## 启动

```bash
cp .env.example .env
# 必须换成真实 NEXA_ADMIN_TOKEN（≥24 位）和 POSTGRES_PASSWORD
# 示例/占位 token 启动会直接拒绝

mkdir -p data/sessions

docker compose -f docker-compose.yml -f docker-compose.build.yml up -d --build
```

- 公开站：`http://127.0.0.1:8080`
- 管理端：`http://127.0.0.1:8081`，令牌是 `NEXA_ADMIN_TOKEN`

也可直接拉预构建镜像：

```bash
docker pull ghcr.io/kiteyuan/nexapulsebot:latest
```

设置 `NEXA_IMAGE=ghcr.io/kiteyuan/nexapulsebot:latest` 后执行 `docker compose up -d` 即可（不必再 `--build`）。

容器内 Admin 监听 `:8081` 是为了端口映射；compose 用 `127.0.0.1:8081:8081` 限制宿主机暴露。这是有意设计。公网请只对 Public `:8080` 做 HTTPS 反代。

**不要提交**：`.env`、`data/sessions/` 下的 Telegram 会话、`data/postgres`。图床 `NEXA_IMAGE_AUTH`、LLM Key 只放在服务器环境或库内设置。

管理端里添加 Telegram 账号（[my.telegram.org](https://my.telegram.org) 的 api_id / api_hash），扫码登录，同步频道，绑定栏目。第一次轮询只记游标，不回灌历史消息。

旧的 Telethon `.session` 不能给 Go 用，需要重新登录。

## 目录

```text
cmd/nexa                 进程入口（组合根）
internal/app             采集/处理调度
internal/collect         采集（Telegram / RSS）
internal/process         规则与按需 LLM
internal/store           PostgreSQL
internal/httpapi         Admin / Public API
internal/ports           窄接口：Accounts/Feeds/Themes/Content、Intake、Pipeline、ImageUploader
internal/kernel          领域模型与 SafeHTTPURL / Present
internal/config          环境变量
internal/imgbed          图床客户端
internal/webui           前端构建结果，嵌入二进制
web/admin                管理端源码
web/public               公开站源码
```

依赖只允许从外往里：`httpapi` / `collect` / `app` 经 `ports` 使用存储与图床，不直接互相耦合。

## 开发

```bash
go run ./cmd/nexa
cd web/admin && npm install && npm run dev
cd web/public && npm install && npm run dev
```

质量闸门：

```bash
make test
make lint
make frontend
```

`internal/webui` 是构建结果，不进仓库。本机要看到页面，先在两个前端目录执行 `npm run build`，或直接用上面的 Docker 构建。

需要本机 PostgreSQL，连接串见 `.env.example`。
