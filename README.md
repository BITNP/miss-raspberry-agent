# miss-raspberry-agent

A QQ bot agent microservice for 网络开拓者协会 (网协): NapCat forwards QQ messages over
WebSocket to the agent, which runs an LLM-powered assistant ("树莓娘") with tools for
replying to messages, scheduling reminders, and maintaining a todo list.

It also ships a small CLI (`cmd/moderator`) that classifies a single comment with a
comment-moderator agent.

## Architecture

```text
QQ (NapCat client)
   │  OneBot11 forward WebSocket server (port 3001)
   ▼
NapCat (mlikiowa/napcat-docker, via docker compose)
   │
   ▼
miss-raspberry-agent (this repo, connects to ws://127.0.0.1:3001)
   │
   ▼
LLM (OpenAI-compatible, e.g. DeepSeek)
```

- **NapCat** is the QQ protocol bridge. It runs as a "WebSocket server" on port `3001`
  and the agent connects to it as a client (`NAPCAT_WS_URL`). NapCat also exposes a
  WebUI on port `6099` for QR login and connection config.
- **The agent** is this Go program. It only talks to NapCat over the WebSocket; it never
  needs direct QQ credentials.

## Requirements

- Go 1.26+
- Docker with the Compose plugin (for NapCat)
- A QQ account (for the bot to log in with)
- An LLM API key (OpenAI-compatible endpoint)

## Configuration

Copy `.env.example` to `.env` and fill in real values. `cmd/server/main.go` loads
`.env` automatically, and `docker compose` reads the same file for interpolation;
do not commit the real `.env` (it is gitignored).

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `MODEL_API_KEY` | yes | — | LLM API key |
| `MODEL_BASE_URL` | no | `https://api.openai.com/v1` | OpenAI-compatible base URL |
| `MODEL_NAME` | no | `gpt-4o-mini` | Model ID |
| `NAPCAT_WS_URL` | no | `ws://127.0.0.1:3001` | NapCat WebSocket server address |
| `NAPCAT_ACCESS_TOKEN` | no | empty | Bearer token if NapCat WS server requires one |
| `NAPCAT_QUICK_ACCOUNT` | no | empty | QQ number NapCat quick-logins with on startup; used by docker compose, not the Go app |
| `HTTP_ADDR` | no | `:4514` | Address the HTTP API listens on |
| `API_TOKEN` | yes | — | Bearer token required by the HTTP API |

## HTTP API

The agent exposes an HTTP API so callers can push messages into the main agent's queue. It
also exposes tagger endpoints for registering in-memory tag sets and tagging text against
them with a tagger agent. It requires a bearer token (`API_TOKEN`). See [`API.md`](API.md) for
the full reference, request/response shapes, and curl examples.

```bash
curl -X POST http://127.0.0.1:4514/api/v1/agents/main/messages \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"platform":"qq","target_id":"10001","content":"你好","context":"初次打招呼"}'
```

## Running locally

### 1. Start NapCat

```bash
docker compose up -d
docker logs -f napcat        # first run prints the WebUI login token
```

- Open <http://localhost:6099/webui>, log in with the token printed in the logs, then
  scan the QR code to log in your QQ account.
- The OneBot11 **WebSocket 服务器** on port `3001` is already provided by the tracked
  `napcat/config/onebot11.json`, so no manual WebUI network setup is needed. Its `token`
  is empty; if you set `NAPCAT_ACCESS_TOKEN`, set the same value there (or in the
  per-account `onebot11_<qq>.json` that NapCat writes).
- **Auto-login after the first QR login.** The QQ session persists in `./napcat/QQ`, so
  set `NAPCAT_QUICK_ACCOUNT` in `.env` to your QQ number and `docker compose up -d`
  again. NapCat then quick-logins that account on every start instead of showing a QR
  code. Keep the value in the gitignored `.env` — never hardcode it in `compose.yaml`,
  `README.md`, or any other tracked file. Leave the variable empty to require a manual
  login. Quick-login tokens can still expire on Tencent's side and force a new QR scan.

On SELinux hosts (e.g. Fedora) the compose bind mounts use the `:z` label so the
container can read `./napcat/`. No further action is needed.

### 2. Configure the agent

```bash
cp .env.example .env
# edit .env: set MODEL_API_KEY (+ MODEL_BASE_URL / MODEL_NAME as needed)
```

### 3. Run the agent

```bash
go run ./cmd/server
```

You should see `[Napcat] Client started`, then `[main] main_agent started, polling its
todo queue...`. Send the bot a private or group message that mentions it (or @ it in a
group) and it should reply.

### Comment moderator CLI

```bash
echo "some comment text" | go run ./cmd/moderator
go run ./cmd/moderator comment.txt
```

### Tests

```bash
go test ./...
go vet ./...
```

## Deploying to a server

The deployment runs the agent binary as a systemd service on the host and NapCat in
Docker, exactly as locally.

### 1. Get the code onto the server

```bash
git clone <repo-url> && cd miss-raspberry-agent
cp .env.example .env   # fill in real secrets
```

Install Go 1.26+ and Docker/Compose on the server if not already present.

### 2. Start NapCat

```bash
docker compose up -d
```

The compose file binds the WebUI (`6099`) and WS port (`3001`) to `127.0.0.1` only.
The first QQ login must happen through the WebUI, so tunnel it over SSH from your
machine:

```bash
ssh -L 6099:127.0.0.1:6099 -L 3001:127.0.0.1:3001 user@server
# then open http://localhost:6099/webui on your machine
```

Complete the QR login as in the local steps; the WebSocket 服务器 on port `3001` comes
from the tracked `napcat/config/onebot11.json`, so no manual network setup is needed. The
QQ session persists in `./napcat/QQ` across restarts.

After that first login, set `NAPCAT_QUICK_ACCOUNT` in the server's `.env` (see
[Configuration](#configuration)) and `docker compose up -d` once more, so NapCat
quick-logins on boot instead of waiting for a QR scan.

### 3. Build the agent

```bash
go build -o /opt/miss-raspberry-agent/bin/agent ./cmd/server
```

### 4. Run the agent as a service

Create `/etc/systemd/system/miss-raspberry-agent.service`:

```ini
[Unit]
Description=miss-raspberry-agent (QQ bot agent)
After=network-online.target docker.service
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=/opt/miss-raspberry-agent
EnvironmentFile=/opt/miss-raspberry-agent/.env
ExecStart=/opt/miss-raspberry-agent/bin/agent
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now miss-raspberry-agent
journalctl -u miss-raspberry-agent -f   # follow logs
```

`EnvironmentFile` injects the variables from `.env`; the binary reads them from the
environment (it also tolerates a `.env` in the working directory).

### 5. Security notes

- Keep ports bound to `127.0.0.1` as in `compose.yaml`; NapCat's WebUI has no strong
  built-in auth and must not be exposed publicly. Use the SSH tunnel instead.
- The tracked `napcat/config/onebot11.json` ships with an empty WS `token`. If you enable
  one, keep it in the gitignored per-account `onebot11_<qq>.json`, not in the tracked file.
- `MODEL_API_KEY` and `NAPCAT_ACCESS_TOKEN` live only in `.env` (gitignored). Never put
  real secrets in committed files.
- Keep the bot's QQ account safe: the session in `./napcat/QQ` is sensitive and is also
  gitignored. Its quick-login credentials live in `napcat/QQ/nt_qq/global/nt_db/login.db`.
- Do not commit your QQ account number. Provide it through the gitignored `.env`
  (`NAPCAT_QUICK_ACCOUNT`) or NapCat's own gitignored `napcat/config/webui.json`
  (`autoLoginAccount`), never through `compose.yaml` or any other tracked file.
