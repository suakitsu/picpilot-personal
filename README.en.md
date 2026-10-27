# PicPilot Personal Go

[简体中文](README.md) | [English](README.en.md) | [日本語](README.ja.md)

Self-hosted PicPilot for personal use. Configure your own image, text and video providers in Settings, then use the workbench, creative studio and infinite canvas. Go handles the business backend; React/Next.js handles the frontend. There is no platform billing, credit storefront or administrator website.

**Private migration acceptance build, not yet declared production-ready.** Do not connect a Java production database. This edition has its own schema and does not automatically import platform users or payment records.

## Quick Start

Install Git, a running Docker Engine / Docker Desktop, and Docker Compose v2. The first build downloads container images and dependencies. Recommended available resources: 4 CPU cores, 8 GB RAM and 10 GB disk space.

```sh
git clone https://github.com/suakitsu/picpilot-personal-go.git
cd picpilot-personal-go
```

Private repository access requires your own GitHub authorization. Never put a token in the clone URL.

Windows PowerShell:

```powershell
./scripts/init.ps1
docker compose up -d --build
docker compose ps
docker compose exec api /picpilot --setup-token
```

Linux / macOS:

```sh
sh scripts/init.sh
docker compose up -d --build
docker compose ps
docker compose exec api /picpilot --setup-token
```

1. Open [PicPilot](http://localhost:8080). Use the setup token displayed in your terminal to create your account. Keep that token private.
2. Sign in and open model settings. Add a provider with its actual compatible protocol, model ID, service URL and your API key.
3. Validate and explicitly enable the configuration. Configuration checks do not invoke paid models. Actual generation can incur charges from your upstream provider.

There are no default accounts, passwords or paid credentials. The initializer generates a random database password; the application generates independent persistent encryption keys. Personal tasks never borrow another user's or an operator's credentials and do not deduct platform credits.

The repository includes its complete `core` source snapshot. **No platform checkout or private module download is needed.** Public registration is disabled; the administrator website is not deployed. Next.js only renders pages and proxies transport. Go owns jobs, business data and provider execution.

## Providers and Local Gateways

Configured adapters cover OpenAI images and Gemini images; OpenAI Chat Completions / Responses for text; APIMart, HFSY and Agnes for video. Choose a protocol compatible with your actual upstream API. A provider's name alone does not establish compatibility.

Inside a container, `127.0.0.1` refers to that container, not your computer. Docker Desktop can reach a host gateway using `host.docker.internal`. Opt into `PICPILOT_ALLOW_PRIVATE_PROVIDERS=true` only when needed. Never expose an unauthenticated local gateway publicly. See [configuration](docs/configuration.md).

## Data and Operations

- HTTP binds only to `127.0.0.1:8080`; PostgreSQL has no published host port. Use HTTPS for public access.
- Provider keys are encrypted server-side and are never returned to browsers or embedded canvases.
- Back up `.env`, the database, uploaded files and `instance-secrets.json` together. Losing the encryption key makes saved provider credentials unreadable.
- Do not stop the system with `docker compose down -v`: that deletes persistent volumes.

```sh
docker compose stop
docker compose start
docker compose logs --tail=100 api
```

[Configuration and HTTPS](docs/configuration.md) · [Backup, restore and upgrades](docs/operations.md) · [Tests](docs/testing.md)

## Editions and License

For multi-user operations, payments and administration, use the separate `suakitsu/picpilot-platform-go` repository. The original Java platform remains independent and unchanged.

Licensed under [AGPL-3.0](LICENSE). Both canvases retain upstream notices and modified corresponding source; see `third_party/*/UPSTREAM.md`. This repository remains private until its owner approves publication. Never commit credentials, images, backups or user data.
