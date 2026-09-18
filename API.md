# API Reference

HTTP API for `miss-raspberry-agent`. The server listens on `HTTP_ADDR` (default `:4514`) and
speaks JSON. All request/response bodies are UTF-8 JSON.

## Authentication

Every `/api/v1/*` endpoint requires a bearer token:

```http
Authorization: Bearer <API_TOKEN>
```

`API_TOKEN` is configured through the environment (see `.env.example`) and is required at
startup. Missing or invalid tokens receive `401 Unauthorized`. Comparison is constant-time.

`GET /healthz` is public.

## Errors

All errors share one shape:

```json
{ "error": "human-readable message" }
```

| Status | Meaning |
| --- | --- |
| `400 Bad Request` | Malformed JSON, missing required field, or invalid field value |
| `401 Unauthorized` | Missing or invalid bearer token |
| `404 Not Found` | Referenced resource (e.g. tag set) does not exist |
| `422 Unprocessable Entity` | Well-formed request but unsupported value (e.g. unknown platform) |
| `500 Internal Server Error` | Unexpected server error |
| `503 Service Unavailable` | Readiness check failed: at least one dependency is down |

## Endpoints

### `GET /healthz`

**Description:** Liveness check. No authentication required.

**Request:** No body, no parameters.

**Response `200 OK`**

```json
{ "status": "ok" }
```

**curl**

```bash
curl http://127.0.0.1:4514/healthz
```

### `GET /api/v1/health`

**Description:** Readiness check for peer services. It reports whether this service can do useful
work: the NapCat (QQ) connection, whether the member directory has been loaded, and how many
items are waiting in the main agent's todo queue. Requires the bearer token. Returns `200 OK`
when every dependency is healthy and `503 Service Unavailable` when any dependency is down.

**Request:** No body, no parameters.

**Response `200 OK` / `503 Service Unavailable`**

| Field | Type | Description |
| --- | --- | --- |
| `status` | string | Aggregate status: `ok` (200) or `degraded` (503). |
| `queue_length` | integer | Number of items waiting for the main agent. Informational only. |
| `dependencies` | array | One entry per probed dependency. |
| `dependencies[].name` | string | Dependency name (`napcat`, `napcat_directory`). |
| `dependencies[].status` | string | `ok` or `degraded`. |
| `dependencies[].error` | string | Reason the dependency is down; omitted when healthy. |

Healthy:

```json
{
  "status": "ok",
  "queue_length": 0,
  "dependencies": [
    { "name": "napcat", "status": "ok" },
    { "name": "napcat_directory", "status": "ok" }
  ]
}
```

Degraded:

```json
{
  "status": "degraded",
  "queue_length": 0,
  "dependencies": [
    { "name": "napcat", "status": "degraded", "error": "no napcat bot connected" },
    { "name": "napcat_directory", "status": "degraded", "error": "member directory not loaded yet" }
  ]
}
```

**Error responses:** `401` (bad token).

**curl**

```bash
curl http://127.0.0.1:4514/api/v1/health \
  -H "Authorization: Bearer $API_TOKEN"
```

### `POST /api/v1/agents/main/messages`

**Description:** Submits a message for the main agent to produce and send. The request is
**queued**: the handler validates it and pushes it onto the main agent's todo list, then returns
immediately. The agent processes the queue asynchronously.

**Request body**

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `platform` | string | yes | Message platform. Currently only `qq` is supported. |
| `target_id` | string | yes | Target identifier on that platform. For `qq`, a numeric user QQ id. |
| `content` | string | yes | The message content the agent should work from. Must be non-blank. |
| `context` | string | no | Extra context to help the agent craft a better reply (e.g. who the user is, prior events). |

> **Platform note:** `qq` currently maps to a **private** QQ chat, so `target_id` is the
> recipient's user QQ number. Group delivery is not exposed through this endpoint yet.

**Response `202 Accepted`**

| Field | Type | Description |
| --- | --- | --- |
| `id` | string | Id of the created todo item. The agent removes it from the queue once handled. |
| `status` | string | Always `queued`. |

```json
{
  "id": "item-1",
  "status": "queued"
}
```

**Error responses:** `400` (malformed body / missing or invalid field), `401` (bad token),
`422` (unsupported `platform`).

**curl**

```bash
curl -X POST http://127.0.0.1:4514/api/v1/agents/main/messages \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"platform":"qq","target_id":"10001","content":"你好","context":"初次打招呼"}'
```

### `POST /api/v1/agents/tagger/tag-sets`

**Description:** Registers (or replaces) a named tag set. Tag sets live in memory: they are
lost when the process restarts. Each set carries a `prompt`: a general instruction describing
the function of the set and the notice the tagger must follow when marking text within it.
Each tag has a `name`, an optional `description`, and an `apply_rule` written in natural
language. Registering a set whose `name` already exists replaces the previous set entirely.
Tag names must be unique within a set.

**Request body**

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `name` | string | yes | Name of the tag set. Non-blank. Used as the key for replacement and later tagging. |
| `prompt` | string | yes | General prompt describing the function of the set and the notice for the tagger when marking text within it. Non-blank. |
| `tags` | array | yes | At least one tag definition. |
| `tags[].name` | string | yes | Tag name. Must be unique within the set. Non-blank. |
| `tags[].description` | string | no | What the tag means. |
| `tags[].apply_rule` | string | yes | Natural-language rule describing when the tag applies. Non-blank. |

```json
{
  "name": "sentiment",
  "prompt": "Tag the text by its sentiment. Only mark text that expresses a clear opinion.",
  "tags": [
    { "name": "positive", "description": "Praise or approval", "apply_rule": "The text expresses praise, approval, or satisfaction." },
    { "name": "negative", "description": "Complaint or disapproval", "apply_rule": "The text expresses criticism, complaint, or dissatisfaction." }
  ]
}
```

**Response `200 OK`**

| Field | Type | Description |
| --- | --- | --- |
| `name` | string | The stored tag set name. |
| `tag_count` | integer | Number of tags stored in the set. |

```json
{ "name": "sentiment", "tag_count": 2 }
```

**Error responses:** `400` (malformed body, missing/blank field, or duplicate tag name
within the set), `401` (bad token).

**curl**

```bash
curl -X POST http://127.0.0.1:4514/api/v1/agents/tagger/tag-sets \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"sentiment","prompt":"Tag the text by its sentiment. Only mark text that expresses a clear opinion.","tags":[{"name":"positive","description":"Praise or approval","apply_rule":"The text expresses praise, approval, or satisfaction."},{"name":"negative","description":"Complaint or disapproval","apply_rule":"The text expresses criticism, complaint, or dissatisfaction."}]}'
```

### `POST /api/v1/agents/tagger/tag`

**Description:** Applies a registered tag set to a piece of text and returns the **single
best-matching tag** chosen by the tagger agent, together with a short reason. When no
registered tag applies, `tag` is `null`.

**Request body**

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `name` | string | yes | Name of a previously registered tag set. Non-blank. |
| `text` | string | yes | The text to tag. Non-blank. |

```json
{ "name": "sentiment", "text": "I really love this update!" }
```

**Response `200 OK`**

| Field | Type | Description |
| --- | --- | --- |
| `name` | string | The tag set used. |
| `tag` | object \| null | The best-matching tag, or `null` when none applies. |
| `tag.name` | string | Matched tag name. |
| `tag.reason` | string | Short reason why the tag applies. |

```json
{
  "name": "sentiment",
  "tag": {
    "name": "positive",
    "reason": "The text expresses strong approval of the update."
  }
}
```

No match:

```json
{ "name": "sentiment", "tag": null }
```

**Error responses:** `400` (malformed body or missing/blank field), `401` (bad token),
`404` (no tag set registered under `name`).

**curl**

```bash
curl -X POST http://127.0.0.1:4514/api/v1/agents/tagger/tag \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"sentiment","text":"I really love this update!"}'
```

## Related configuration

| Variable | Default | Description |
| --- | --- | --- |
| `HTTP_ADDR` | `:4514` | Address the HTTP API listens on |
| `API_TOKEN` | — (required) | Bearer token for `/api/v1/*` |
