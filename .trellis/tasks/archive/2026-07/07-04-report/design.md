# Report Streaming Output Design

## Architecture

The implementation uses report events as the public streaming boundary. Document records concise generation delta events while it consumes AI Gateway streaming responses. Document also exposes a report-level SSE route that replays newly persisted report events. Gateway proxies that route as a public SSE endpoint. The frontend subscribes through the existing Gateway stream helper and renders a live preview while the existing polling path continues to hydrate final outline and section data.

This keeps ownership aligned:

- AI Gateway owns provider streaming and returns OpenAI-compatible chunks.
- Document owns prompts, report parsing, job events, final persistence, and degradation semantics.
- Gateway owns browser route exposure, auth context propagation, and SSE proxying.
- Frontend owns stream subscription, cancellation, preview display, and fallback to polling.

## Data Flow

```text
AI provider
  -> AI Gateway /internal/v1/chat/completions stream
  -> Document streaming chat client
  -> ReportGenerationService accumulates text and records delta events
  -> report_events table
  -> Document /reports/{reportId}/events/stream SSE
  -> Gateway /api/v1/reports/{reportId}/events/stream SSE
  -> Report generation page live preview
```

Polling remains:

```text
frontend -> Gateway /api/v1/report-jobs/{jobId}
frontend -> Gateway /api/v1/reports/{reportId}/events
frontend -> Gateway /api/v1/reports/{reportId}/outlines and /sections
```

## Event Contract

The SSE route emits ordinary `ReportEvent` JSON in each frame:

```text
event: report.event
data: {"id":"...","reportId":"...","jobId":"...","eventType":"outline.delta","message":"...","createdAt":"..."}
```

Delta events are intentionally small because the current durable event schema has only `message text` and existing sanitization compacts messages. Event types:

- `outline.delta`: incremental text from outline generation.
- `section.delta`: incremental text from section generation.
- Existing terminal / progress events remain unchanged.

For `section.delta`, the message uses a compact JSON object when section attribution is needed:

```json
{"sectionId":"...","text":"..."}
```

If parsing that compact JSON fails on the frontend, the message is treated as plain text.

## Backend Design

Document HTTP adds `GET /reports/{reportId}/events/stream`.

- Validates `jobSvc` availability.
- Sets `Content-Type: text/event-stream`, `Cache-Control: no-cache`, and `X-Accel-Buffering: no`.
- Uses `http.Flusher`.
- Polls `ListEvents` periodically and emits unseen events in chronological order.
- Emits heartbeat comments while idle.
- Stops when the client context is canceled.

Document AI Gateway client adds a streaming method:

```go
StreamChatCompletion(ctx, reqCtx, input, onDelta) (ChatCompletionResponse, error)
```

- Sends `stream: true`.
- Sends `Accept: text/event-stream`.
- Parses OpenAI-compatible SSE chunks.
- Calls `onDelta` for `choices[].delta.content`.
- Accumulates all deltas to return the same `ChatCompletionResponse` shape used by existing parsing code.
- Treats malformed stream chunks as dependency errors.

ReportGenerationService prefers streaming chat through the same interface and falls back only when the concrete client does not implement the streaming method. During streaming, it records concise `outline.delta` / `section.delta` events and still parses the final accumulated JSON exactly like current non-streaming generation.

## Gateway Design

Gateway adds an active route:

```text
GET /api/v1/reports/{reportId}/events/stream
```

The route has owner `document`, `StreamResponse: true`, and an OpenAPI `text/event-stream` success response. Gateway SSE contract tests are updated so this route is recognized alongside QA streaming.

The proxy timeout bypass recognizes this route as an SSE route when the request accepts `text/event-stream`.

## Frontend Design

Add a report streaming hook under `apps/web/src/features/reports/hooks/`.

The hook:

- Uses `streamGateway(path, { method: 'GET' })`.
- Parses `report.event` frames into `ReportEvent`.
- Appends `outline.delta` and `section.delta` text to local preview state.
- Exposes `status`, `outlineText`, `sectionTextById`, `error`, and `abort`.
- Invalidates or lets existing polling refresh final state when terminal events arrive.
- Ignores stream errors for generation correctness and lets polling continue.

The report generation page opens the stream while an outline/content job is active and renders a compact live output preview near the job progress panel. Final editable outline and sections still come from persisted resources.

## Compatibility And Rollback

The new SSE route is additive. Existing polling clients continue to work. If the frontend stream fails or a browser lacks readable stream support, polling still shows final job progress and results. If server-side streaming AI fails, the job follows existing error paths. A rollback can remove the route and hook without changing existing report job contracts.

## Security

Streamed events contain only sanitized report event fields. They must not contain prompts, provider raw errors, provider credentials, internal service URLs, object keys, raw knowledge payloads, or private tool data. Knowledge retrieval warnings remain high-level.
