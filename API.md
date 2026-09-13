# API Reference

HTTP API for `miss-raspberry-agent`. The server listens on `HTTP_ADDR` (default `:8080`) and
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
| `501 Not Implemented` | Endpoint accepted the request but its backing feature is not built yet |

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
curl http://127.0.0.1:8080/healthz
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
curl -X POST http://127.0.0.1:8080/api/v1/agents/main/messages \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"platform":"qq","target_id":"10001","content":"你好","context":"初次打招呼"}'
```

### `POST /api/v1/agents/tagger/tag-sets`

**Description:** Registers (or replaces) a named tag set. Tag sets live in memory: they are
lost when the process restarts. Each tag has a `name`, an optional `description`, and an
`apply_rule` written in natural language. Registering a set whose `name` already exists
replaces the previous set entirely. Tag names must be unique within a set.

**Request body**

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `name` | string | yes | Name of the tag set. Non-blank. Used as the key for replacement and later tagging. |
| `tags` | array | yes | At least one tag definition. |
| `tags[].name` | string | yes | Tag name. Must be unique within the set. Non-blank. |
| `tags[].description` | string | no | What the tag means. |
| `tags[].apply_rule` | string | yes | Natural-language rule describing when the tag applies. Non-blank. |

```json
{
  "name": "sentiment",
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
curl -X POST http://127.0.0.1:8080/api/v1/agents/tagger/tag-sets \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"sentiment","tags":[{"name":"positive","description":"Praise or approval","apply_rule":"The text expresses praise, approval, or satisfaction."},{"name":"negative","description":"Complaint or disapproval","apply_rule":"The text expresses criticism, complaint, or dissatisfaction."}]}'
```

### `POST /api/v1/agents/tagger/tag`

**Description:** Applies a registered tag set to a piece of text and returns the tags whose
`apply_rule` matches. The tagger agent is **not built yet**: the request is validated and the
tag set is looked up, but the endpoint currently responds `501 Not Implemented`.

**Request body**

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `name` | string | yes | Name of a previously registered tag set. Non-blank. |
| `text` | string | yes | The text to tag. Non-blank. |

```json
{ "name": "sentiment", "text": "I really love this update!" }
```

**Response `200 OK`** (once implemented)

| Field | Type | Description |
| --- | --- | --- |
| `name` | string | The tag set used. |
| `tags` | array | Tags that apply to `text`. |
| `tags[].name` | string | Matched tag name. |
| `tags[].description` | string | Matched tag description, if any. |

**Error responses:** `400` (malformed body or missing/blank field), `401` (bad token),
`404` (no tag set registered under `name`), `501` (tagger not implemented yet).

**curl**

```bash
curl -X POST http://127.0.0.1:8080/api/v1/agents/tagger/tag \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"sentiment","text":"I really love this update!"}'
```

## Related configuration

| Variable | Default | Description |
| --- | --- | --- |
| `HTTP_ADDR` | `:8080` | Address the HTTP API listens on |
| `API_TOKEN` | — (required) | Bearer token for `/api/v1/*` |
