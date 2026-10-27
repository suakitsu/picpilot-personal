# PicPilot Personal Go

[简体中文](README.md) | [English](README.en.md) | [日本語](README.ja.md)

自托管的 PicPilot 个人版。用户在自己的设置页配置图片、文本和视频 Provider，在工作台、创作工作台及自由画布中使用。业务后端为 Go，前端为 React/Next.js；不包含平台充值、积分售卖和管理员站点。

**当前为私有迁移验收版本，尚未宣布正式可用。** 不要连接 Java 版生产数据库。个人版使用独立数据库结构，不会自动迁移原平台的用户或充值数据。

## 快速启动

需要 Git、运行中的 Docker Engine / Docker Desktop 和 Docker Compose v2。首次构建需要联网下载镜像及依赖。建议至少 4 核、8 GB 可用内存和 10 GB 可用磁盘。

```sh
git clone https://github.com/suakitsu/picpilot-personal-go.git
cd picpilot-personal-go
```

仓库私有期间需要自己的 GitHub 读取权限；不要把 Token 放进克隆地址。

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

## 模型与本机网关

图片支持配置的 OpenAI 图片及 Gemini 适配器，文本支持 OpenAI Chat Completions / Responses，视频支持配置的 APIMart、HFSY、Agnes 适配器。必须选择与上游真实接口兼容的协议；任意厂商接口不能仅靠改名字通用。

若模型网关运行在 Docker 宿主机上，容器里的 `127.0.0.1` 指的是容器本身。Docker Desktop 可使用 `host.docker.internal`，并在确有需要时设置 `PICPILOT_ALLOW_PRIVATE_PROVIDERS=true`。不要向公网暴露无认证的本机网关。其他部署方式见 [配置说明](docs/configuration.md)。

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

## 版本与许可

需要运营多用户平台、充值或管理员后台时，使用独立的 `suakitsu/picpilot-platform-go`。原 Java 平台保持独立，不被本次迁移修改。

本项目采用 [AGPL-3.0](LICENSE)。两个画布保留上游许可和修改后的对应源码，见 `third_party/*/UPSTREAM.md`。当前仓库保持私有，公开发布需等待所有者确认。请勿向 Git 提交密钥、图片、备份或用户数据。
