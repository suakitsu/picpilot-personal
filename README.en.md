# PicPilot Personal

[简体中文](README.md) | [English](README.en.md) | [日本語](README.ja.md)

An open-source personal AI workspace for self-hosting: image generation, video generation, e-commerce creation, reference images, creative studio and infinite canvas. Connect your own image, text and video providers. There is no platform billing, credit storefront or administrator website.

> **Status: public migration acceptance build.** Deploy with a new database; do not connect the Java production database.

## Quick start

Install Git, Docker Engine/Docker Desktop and Docker Compose v2. The first build is most comfortable with 4 CPU cores, 8 GB RAM and 10 GB free disk.

```sh
git clone https://github.com/suakitsu/picpilot-personal.git
cd picpilot-personal
```

Windows PowerShell:

```powershell
./scripts/init.ps1
docker compose up -d --build
docker compose ps
docker compose exec api /picpilot --setup-token
```

Linux / macOS: replace the first command with `sh scripts/init.sh` and keep the rest.

1. Open `http://localhost:8080` and use the one-time token printed in the terminal to create your personal account.
2. Open model settings and add a provider with its real protocol, model, URL and API key.
3. Validate and enable it, then create in the workbench, creative studio or infinite canvas.

There are no default accounts, passwords or platform keys. Personal edition tasks use only the current user's providers. Provider keys are encrypted server-side and are never returned to the browser or canvas.

## Providers and local gateways

Configured provider base URLs support both `http://` and `https://`, including local gateways such as `http://127.0.0.1:8888/v1`. From a container, Docker Desktop can reach a host gateway through `host.docker.internal`. User-submitted external image references still use the strict safe-download policy.

The provider must match the upstream protocol. A vendor name alone does not make an arbitrary API compatible. Configuration checks do not call paid models; actual generation costs are controlled by the upstream service.

## Screenshots

![Personal provider settings](docs/screenshots/personal-providers-desktop.png)

![Personal generated result example](docs/screenshots/personal-sualoom-real-lolita.jpg)

## Maintenance

- By default, only `127.0.0.1:8080` is published; PostgreSQL has no host port.
- For public access, configure HTTPS at the gateway or reverse proxy and set the correct public URL.
- Back up `.env`, the database, uploads and `instance-secrets.json`; never commit them.
- Use `docker compose stop` for routine shutdown. `docker compose down -v` deletes persistent volumes.

More: [Configuration and HTTPS](docs/configuration.md) · [Backup, restore and upgrades](docs/operations.md) · [Tests](docs/testing.md)

## Editions and license

For multi-user operations, payments or administration, use the separate [`picpilot-platform`](https://github.com/suakitsu/picpilot-platform) repository. The original Java platform remains independent.

PicPilot-owned code is available under [MIT](LICENSE). Bundled Nova data and both canvas projects retain AGPL-3.0; the combined application is distributed under AGPL-3.0. See [LICENSING.md](LICENSING.md) for scope and third-party notices.
