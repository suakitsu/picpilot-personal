# PicPilot Personal

[简体中文](README.md) | [English](README.en.md) | [日本語](README.ja.md)

面向个人自托管的开源 AI 创作工作台，支持图片生成、视频生成、电商创作、参考图、创意工作台和自由画布。连接自己的图片、文本和视频 Provider；不包含平台计费、积分商城或管理员网站。

> **当前状态：公开迁移验收版。** 使用新数据库部署，不要连接 Java 版生产数据库。

## 快速启动

需要 Git、Docker Engine/Docker Desktop 和 Docker Compose v2。首次构建建议准备 4 核 CPU、8 GB 内存和 10 GB 磁盘。

```sh
git clone https://github.com/suakitsu/picpilot-personal.git
cd picpilot-personal
```

Windows PowerShell：

```powershell
./scripts/init.ps1
docker compose up -d --build
docker compose ps
docker compose exec api /picpilot --setup-token
```

Linux / macOS：把第一行替换为 `sh scripts/init.sh`，后续命令相同。

1. 打开 `http://localhost:8080`，用终端显示的一次性令牌创建个人账号。
2. 登录后打开模型设置，填写 Provider 的真实协议、模型、地址和 API Key。
3. 验证配置并启用，然后在工作台、创意工作台或自由画布中开始创作。

没有默认账号、密码或平台密钥。个人版只使用当前用户自己配置的 Provider，API Key 在服务端加密保存，不会返回给浏览器或画布。

## Provider 与本地网关

已配置的 Provider Base URL 同时支持 `http://` 和 `https://`，也可以使用 `http://127.0.0.1:8888/v1` 这类本机网关。容器访问宿主机网关时，Docker Desktop 使用 `host.docker.internal`。用户提交的外部图片引用仍使用严格安全下载策略。

Provider 必须与上游真实协议兼容；填写厂商名称不会自动兼容任意 API。配置检查不会调用付费模型，实际生成费用由上游服务决定。

## 界面预览

![个人版 Provider 设置](docs/screenshots/personal-providers-desktop.png)

![个人版生成结果示例](docs/screenshots/personal-sualoom-real-lolita.jpg)

## 维护

- 默认仅监听 `127.0.0.1:8080`，PostgreSQL 不开放宿主机端口。
- 对公网提供服务时，请在网关或反向代理层配置 HTTPS，并设置正确的公开 URL。
- 备份 `.env`、数据库、上传文件和 `instance-secrets.json`；不要把它们提交到 Git。
- 日常停机使用 `docker compose stop`。`docker compose down -v` 会删除数据卷。

更多内容：[配置与 HTTPS](docs/configuration.md) · [备份、恢复与升级](docs/operations.md) · [测试](docs/testing.md)

## 版本关系与许可

需要多用户运营、充值或管理员后台时，请使用独立的 [`picpilot-platform`](https://github.com/suakitsu/picpilot-platform)。原 Java 平台保持独立。

PicPilot 自有代码采用 [MIT](LICENSE)。内置的 Nova 数据和两个画布项目保留 AGPL-3.0；包含这些内容的组合应用按 AGPL-3.0 发布。完整范围和第三方清单见 [LICENSING.md](LICENSING.md)。
