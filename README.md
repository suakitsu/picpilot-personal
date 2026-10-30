# PicPilot Personal

[简体中文](README.md) | [English](README.en.md) | [日本語](README.ja.md)

面向创作者的自托管 PicPilot 个人 AI 创作工作台，支持图片生成、视频生成、电商创作、参考图工作流与自由画布。用户在自己的设置页连接图片、文本和视频 Provider，完成灵感探索和视觉内容制作。业务后端为 Go，前端为 React/Next.js；不包含平台充值、积分售卖和管理员站点。

**当前为已公开的迁移验收版本，尚未宣布正式可用。** 不要连接 Java 版生产数据库。个人版使用独立数据库结构，不会自动迁移原平台的用户或充值数据。

## 快速启动

需要 Git，以及 Docker Engine/Docker Compose v2，或本机 Go、Node.js 和 PostgreSQL。Docker 只是打包启动方式，不是运行前提。首次构建需要联网下载镜像及依赖。建议至少 4 核、8 GB 可用内存和 10 GB 可用磁盘。

```sh
git clone https://github.com/suakitsu/picpilot-personal.git
cd picpilot-personal
```

仓库已公开，克隆无需 GitHub 令牌。

Windows PowerShell：

```powershell
./scripts/init.ps1
docker compose up -d --build
docker compose ps
docker compose exec api /picpilot --setup-token
```

Linux / macOS：

```sh
sh scripts/init.sh
docker compose up -d --build
docker compose ps
docker compose exec api /picpilot --setup-token
```

1. 打开 [PicPilot](http://localhost:8080)，使用终端显示的初始化令牌创建自己的账号。不要分享令牌。
2. 登录后进入设置中的模型配置，添加图片、文本或视频服务，填写兼容协议、模型 ID、服务地址和自己的 API Key。
3. 检查并明确启用配置。配置检查本身不会生成图片或调用付费模型；提交真实生成任务后，上游可能按自己的规则收费。

没有默认账号、默认密码或预置付费密钥。初始化脚本生成随机数据库密码，应用在数据卷内生成独立加密密钥。个人任务不会借用平台或其他用户的 Provider 配置，也不会扣除平台积分。

本仓库包含完整的 `core` 源码快照，**无需克隆平台版或下载私有模块**。个人版不启动管理员站点，也不开放公众注册。Next.js 仅用于前端和传输代理；任务、业务数据和模型调用由 Go 服务处理。

## 冷启动验收

新用户只需要一个仓库和 Docker，不需要安装 Go、Node 或 PostgreSQL：

```powershell
./scripts/init.ps1                 # 只需首次执行，生成 .env 和随机数据库密码
docker compose up -d --build       # 构建 Go API、用户站点和网关
docker compose ps                   # 等待 db/api/user/gateway 为 running/healthy
docker compose exec api /picpilot --setup-token
```

初始化令牌只在终端显示一次。打开 `http://localhost:8080`，使用令牌创建个人账号；看到首页或登录页，
并且 API 通过 Compose 健康检查，就表示启动完成。Linux/macOS 使用 `sh scripts/init.sh` 替代第一行。
首次构建需要下载镜像和依赖，速度取决于网络；后续启动使用 `docker compose start`，不需要重新构建。

独立空数据库验收已覆盖：镜像构建、迁移、首次账号、登录、个人 Provider 加密保存、Mock 图片/文案任务、
幂等、签名文件和退出登录。下图来自本地浏览器验收，不包含真实用户数据，也不会调用付费模型。

以下是之前从完整桌面浏览器页面截取的 1440px 验收图，不是移动端视口或窄长裁剪图：

![个人版工作台](docs/screenshots/personal-workbench-desktop.png)

![个人版模型配置](docs/screenshots/personal-providers-desktop.png)

### 真实 Provider 实测展示（2026-10-30）

在独立本机数据目录中由个人用户配置了 OpenAI Images 兼容的 Sualoom Provider（`gpt-image-2`），通过个人版自由创作实际生成一张 1K 方形的成年女性 Lolita 时装摄影图。下面的截图来自真实成功任务的成品详情页，和上面的 Mock 验收图分开；这不是默认测试步骤，上游服务可能按其规则计费。

![个人版 Sualoom gpt-image-2 真实 Lolita 成品](docs/screenshots/personal-sualoom-real-lolita.jpg)

### 不使用 Docker：Go 直启验收

没有 Docker 时可以直接启动 Go API 和个人版 Next.js 前端。下面的示例把个人数据、Go 编译缓存和 npm 缓存都放到 `E:\picpilot`；PostgreSQL 使用你准备好的独立数据库，不要连接生产库。

首次克隆或上游画布源码更新后，先构建两个被忽略的静态画布目录：

```powershell
./scripts/build-canvas-direct.ps1
```

脚本只生成本机 `third_party/nova-image-studio/frontend/out` 和 `third_party/infinite-canvas/web/dist`，不写入 Git；后续只改 PicPilot Go/Next 代码时不用重复构建。

```powershell
$env:DATABASE_URL = "postgres://picpilot:<password>@127.0.0.1:5432/picpilot?sslmode=disable"
$env:PICPILOT_DATA_DIR = "E:\picpilot\runtime\personal-data"
$env:PICPILOT_LISTEN = "127.0.0.1:18092"
$env:PICPILOT_PUBLIC_URL = "http://localhost:18093"
$env:PICPILOT_ALLOWED_ORIGINS = "http://localhost:18093"
$env:PICPILOT_DEMO_MODE = "true"       # 仅本地 Mock 验收；真实部署必须改为 false
# Provider Base URL 同时支持 http:// 和 https://；探测/实际请求是否可达由上游网络决定
$env:GOCACHE = "E:\picpilot\development-cache\go\build"
$env:GOMODCACHE = "E:\picpilot\development-cache\go\modules"
go run ./cmd/picpilot
```

另开一个 PowerShell 窗口启动前端：

```powershell
cd web\user
$env:GO_API_INTERNAL_URL = "http://127.0.0.1:18092"
npm run dev -- --hostname 127.0.0.1 --port 18093
```

打开 `http://localhost:18093` 完成初始化和登录。Go API 会自动执行迁移。`PICPILOT_DEMO_MODE=true` 只允许 Mock 图片/文案验收，不会调用付费模型；正式部署时使用独立数据库、HTTPS 和 `PICPILOT_DEMO_MODE=false`。

启动排查：

- `POSTGRES_PASSWORD` 缺失：先运行 `init.ps1`/`init.sh`，不要把真实密钥写进 `.env.example`。
- Docker Engine 不可用：可改用上面的 Go 直启路径；若使用 Docker，再启动 Docker Desktop 并确认 `docker info` 返回 Server 信息。
- 端口冲突：在 `.env` 修改 `PICPILOT_HTTP_PORT` 和 `PICPILOT_PUBLIC_URL`，保持两者一致。
- 页面打不开：查看 `docker compose ps` 和 `docker compose logs --tail=100 api gateway`，不要先删卷。
- 日常停机用 `docker compose stop`，恢复用 `docker compose start`；`down -v` 会删除数据库和应用数据。

详细备份、恢复、升级和回滚步骤见 [运维文档](docs/operations.md)，测试边界见 [测试文档](docs/testing.md)。

## 模型与本机网关

图片支持配置的 OpenAI 图片及 Gemini 适配器，文本支持 OpenAI Chat Completions / Responses，视频支持配置的 APIMart、HFSY、Agnes 适配器。必须选择与上游真实接口兼容的协议；任意厂商接口不能仅靠改名字通用。

已配置的 Provider Base URL 同时支持 HTTP 和 HTTPS，`http://127.0.0.1:8888/v1` 这类本机网关也可以直接配置。若模型网关运行在 Docker 宿主机上，容器里的 `127.0.0.1` 指的是容器本身，Docker Desktop 可使用 `host.docker.internal`。不要向公网暴露无认证的本机网关。其他部署方式见 [配置说明](docs/configuration.md)。

## 数据与维护

- 默认仅监听 `127.0.0.1:8080`，数据库不开放宿主机端口。公开访问应启用 HTTPS。
- API Key 加密保存在服务端，不返回给浏览器或嵌入画布。
- `.env`、上传文件、数据库及 `instance-secrets.json` 必须一起妥善备份。丢失加密密钥会使已保存的 API Key 无法解密。
- 不要用 `docker compose down -v` 停机，该命令会删除数据卷。

```sh
docker compose stop
docker compose start
docker compose logs --tail=100 api
```

[配置与 HTTPS](docs/configuration.md) · [备份、恢复与升级](docs/operations.md) · [测试](docs/testing.md)

## 鸣谢

感谢以下开源项目提供画布能力。个人版保留其许可证、对应源码和来源说明，并在此基础上接入 PicPilot 的品牌、认证、任务桥接和个人存储：

- [Nova Image Studio](https://github.com/tianjiangqiji/nova-image-studio)：创作工作台，嵌入版本 `3.1.3`，固定提交 `7768f3f8d7f47e04c6d18572837a086c7a533161`，许可证 AGPL-3.0。
- [Infinite Canvas](https://github.com/basketikun/infinite-canvas)：自由画布，固定提交 `705fdd9b2f7bfe6fd58156fd5c1cdb1ad251c767`，许可证 GNU AGPL v3。

许可证、来源和修改说明见 `third_party/nova-image-studio/UPSTREAM.md` 与 `third_party/infinite-canvas/PROVENANCE.txt`。

## 版本与许可

需要运营多用户平台、充值或管理员后台时，使用独立的 `suakitsu/picpilot-platform`。原 Java 平台保持独立，不被本次迁移修改。

PicPilot 自有原创代码采用 [MIT](LICENSE) 许可。内置第三方画布及 `core/nova/defaults/` 中的 Nova 默认数据保留 AGPL-3.0；包含这些内容的组合应用仍按 AGPL-3.0 发布，不能视为纯 MIT 项目。许可范围、第三方清单及对应源码义务见 [许可说明](LICENSING.md)。
