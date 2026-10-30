# Configuration

Use `.env` created by `scripts/init.ps1` or `scripts/init.sh`. It is ignored by Git. Never paste its contents into an issue, screenshot, log, or chat. Restart the affected services after changing runtime configuration:

```sh
docker compose up -d
```

Do not run `docker compose config` without `--quiet` when sharing output: Compose renders secrets into its expanded configuration.

## First Run and Origins

| Setting | Default | Meaning |
| --- | --- | --- |
| `COMPOSE_PROJECT_NAME` | `picpilot-platform` or `picpilot-personal` | Namespace for this installation's containers and volumes |
| `POSTGRES_PASSWORD` | Random initializer output | Database password; never a sample or shared password |
| `PICPILOT_PUBLIC_URL` | `http://localhost:8080` | Exact user-site origin, including a nonstandard port |
| `PICPILOT_BIND_ADDRESS` | `127.0.0.1` | Local-only HTTP binding |
| `PICPILOT_HTTP_PORT` | `8080` | User-site host port |
| `PICPILOT_NETWORK_SUBNET` | `172.28.58.0/24` | Dedicated application network; choose a free subnet if it conflicts |
| `PICPILOT_PROXY_ADDRESS` | `172.28.58.10` | Proxy address within that subnet; only this proxy's forwarding headers are trusted |
| `PICPILOT_REGISTRATION_ENABLED` | `false` | Platform registration only; personal never enables public registration |
| `PICPILOT_DEMO_MODE` | `false` | Synthetic, clearly labeled mock image/text/video results; not real AI and not a production billing mode |

Public origins must be HTTP(S) origins without credentials, paths, fragments, or query strings. A changed origin requires matching browser URLs, allowed origins, OAuth callback and certificate configuration. If running two installations simultaneously, use distinct project names, ports, subnets and proxy addresses. They must not share a database or application volume.

The Compose file supplies `DATABASE_URL`, `PICPILOT_LISTEN=0.0.0.0:8080`, `PICPILOT_DATA_DIR=/data`, allowed origins and trusted-proxy CIDRs. PostgreSQL is private to the installation. Do not publish its port to make the UI work.

## HTTPS

For access over the internet, point a DNS name at your server, allow TCP 80/443, and set:

```dotenv
PICPILOT_PUBLIC_URL=https://picpilot.example.com
```

Replace the example name with a domain you control. Start Caddy's HTTPS configuration:

```sh
docker compose -f compose.yml -f compose.https.yml up -d --build
```

Caddy obtains and renews certificates. Its certificate volumes are persistent. Do not put another untrusted reverse proxy in front without configuring and verifying the forwarding chain. The default loopback HTTP port bindings are not a substitute for TLS on the public internet.

Both canvases are built from the checked-in frontend sources and served under the user origin at `/canvas/nova/` and `/canvas/infinite/`. They do not need separate domains, registrations, API keys, Node business servers or Python agent processes. Keep executable plugin code isolated or disabled in this embedding; do not relax CSP to add `unsafe-eval` as a workaround.

## Model Providers

Use the signed-in user's Provider settings page. Keys are encrypted with an instance key; responses report only whether a key exists. Leaving a key blank preserves the previous key; explicit removal clears it. Personal edition has no platform operator credentials or paid credit ledger.

Supported protocol families are enumerated by the settings API:

- Text: OpenAI chat completions and Responses.
- Image: OpenAI image generations/edits and Gemini generate-content adapters.
- Video: APIMart, HFSY and Agnes adapters, with their configured model and resolution mappings.
- Local Worker: the four OpenAI v1 protocols in the authoritative protocol document; image model `gpt-image-2`.

Use the real upstream model ID and compatible adapter profile. An OpenAI-compatible label does not guarantee all parameters or endpoints. Model credentials stay on the server; they are not sent to embedded canvases.

Priority orders enabled providers. Provider concurrency is coordinated in PostgreSQL, not only in an individual Go process. HTTP adapters never blindly repeat a generation POST after an ambiguous network failure. A task with uncertain submission is not automatically sent to another paid provider merely because a timeout occurred.

HTTP circuit defaults are one qualifying failure, 1,800 seconds for authentication failures, and 600 seconds for transient image/video failures or 120 seconds for text. Per-provider `settings.circuit_breaker` can set `failure_threshold` (1-20), `auth_seconds` (0-86400), and `transient_seconds` (0-86400). These are circuit cooldowns, not task execution timeouts. Validation rejections and user cancellation must not be interpreted as upstream availability failures.

## Local Worker

Read [Local Worker Protocol v1](local-worker-protocol-v1.md), not the generic browser API envelope. The server does not need the worker's local Sub2API/CLIProxyAPI URL, key, or OAuth tokens.

1. Deploy the server over HTTPS and leave both seeded local providers disabled.
2. Create a user-scoped Worker Token through the personal owner configuration. The API namespace is `/api/me/ai-config`, not `/api/admin/ai-config`. It is displayed once; the database stores a hash. Copy it privately to the worker. Never add it to `.env.example`, source or test fixtures.
3. Bind the intended worker ID and confirm a heartbeat and an empty long poll.
4. Set image/text concurrency to match the worker's advertised capacity, then enable one local provider at a time.

`PICPILOT_WORKER_LEASE_GRACE_SECONDS` defaults to 15 (5-120); `PICPILOT_WORKER_INPUT_TTL_SECONDS` defaults to 300 (30-600). Normal installations use independent randomly generated Worker pepper and signing keys in `instance-secrets.json`.

Explicit environment overrides are `PICPILOT_AUTH_KEY`, `PICPILOT_ENCRYPTION_KEY`, `PICPILOT_WORKER_TOKEN_PEPPER`, `PICPILOT_WORKER_SIGNING_SECRET`, and `PICPILOT_SETUP_TOKEN`. The four cryptographic keys must each be independent, unpadded URL-safe base64 encodings of 32 random bytes. Prefer generated persistent keys. Changing them casually invalidates credentials, sessions, tokens or signed links. Back up any external secret-manager values separately if they are not in `.env`.

The historical IDs `local-cliproxy-image` and `local-cliproxy-text` are stable provider identifiers, not a requirement to run CLIProxyAPI. The server never starts a process on the worker machine.

## Email, OAuth and CAPTCHA

All are optional and disabled without configuration. Never use live services in automated tests.

SMTP uses `PICPILOT_SMTP_HOST`, `PICPILOT_SMTP_PORT` (587), `PICPILOT_SMTP_USERNAME`, `PICPILOT_SMTP_PASSWORD`, `PICPILOT_SMTP_FROM`, and `PICPILOT_SMTP_TLS_MODE` (`starttls` or `tls`). Certificate validation is mandatory. Without SMTP, password login and local owner recovery work; email-code flows remain unavailable.

Platform GitHub OAuth uses `PICPILOT_GITHUB_CLIENT_ID` and `PICPILOT_GITHUB_CLIENT_SECRET`. Register `${PICPILOT_PUBLIC_URL}/api/auth/github/callback` as its callback. Both values must be supplied together. Verified GitHub email does not silently attach to an existing local account.

Turnstile uses `PICPILOT_TURNSTILE_SITE_KEY` and `PICPILOT_TURNSTILE_SECRET` together. Configure both your user and admin hostnames at the service. Only explicitly configured UI hostnames are accepted in verification responses.

Personal edition does not expose platform payment channels, plans, credit adjustments or administrator APIs. Configuring a model provider does not enable platform payments.
