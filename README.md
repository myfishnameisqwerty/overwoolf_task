# Real-Time Event Tracking Platform

Embeddable event tracking SDK. A small `<script>` on any page collects pageviews, clicks, and custom events and ships them to a Go backend (SQLite + WAL). A built-in dashboard shows live sessions and event breakdowns.

## Quick Start

```bash
git clone https://github.com/myfishnameisqwerty/overwoolf_task.git
cd overwoolf_task
docker compose up --build
```

Backend on `http://localhost:8080`. Schema is embedded and applied on startup; `data/` is a Docker volume. New `clientId`s auto-register with defaults on first `/config` hit, which the widget calls during initialization - no manual setup.

Reset state: `docker compose down && rm -rf data/ && docker compose up`.

## Embed the widget

Paste into the page's `<head>`. The same snippet works on any site; events are grouped by the page's domain.

```html
<script>
  window._tracker = window._tracker || [];
  (function() {
    var s = document.createElement('script');
    s.src = 'http://localhost:8080/widget.js';
    s.async = true;
    document.head.appendChild(s);
  })();
</script>
```

The widget then:
- Fires a `pageview` on load.
- Fires `click` on any element with `data-track="<label>"`.
- Batches and flushes every 5 s (configurable per client via `/config`).
- Renders a Shadow-DOM-isolated badge with the live active-viewer count.

Widget API on `window._tracker`:

| Call | Effect |
|---|---|
| `_tracker.push(['track', 'name', { ...meta }])` | Fire a custom event. Calls made before the bundle loads are queued and replayed. |
| `_tracker.destroy()` | Remove the badge, cancel timers, detach listeners. |

## Dashboard

Two ways to open it - pick whichever:

```
http://localhost:8080/dashboard      # served by the backend, zero extra tooling
```

The dashboard auto-detects which port it's running on and routes its API calls to `:8080`. Three panels: Session List (auto-refreshes every 5 s), Session Detail (click a row), Event Breakdown (by `clientId`).

![example](image.png)

## Test page

[`test-site.html`](test-site.html) is a static demo with the widget embedded:

| Server | Command | URL |
|---|---|---|
| **VS Code Live Server** | Install the [extension](https://marketplace.visualstudio.com/items?itemName=ritwickdey.LiveServer), right-click → "Open with Live Server" | `http://localhost:5500/test-site.html` |
| **Python** | `python3 -m http.server 5500` | `http://localhost:5500/test-site.html` |
| **Node** | `npx serve -l 5500` | `http://localhost:5500/test-site.html` |

Or paste the embed snippet into the DevTools console on any open tab - no server needed.

## API

| Method | Path | Returns |
|--------|------|---|
| GET  | `/config/:clientId` | `{ clientId, flushInterval, trackedEvents }` |
| POST | `/events` | `null` (200 on success; `eventId` required per event) |
| GET  | `/sessions` | Active sessions in last 5 min, [paginated](#pagination) |
| GET  | `/sessions/:sessionId` | Event history for one session, [paginated](#pagination) |
| GET  | `/stats/:clientId` | `{ sessionCount, breakdown: { type: count } }` last 5 min |
| GET  | `/widget.js` | Widget bundle |
| GET  | `/dashboard` | Dashboard HTML |

All JSON responses use the envelope `{ "status": "success" \| "error", "data": ... }`.

### Pagination

Sessions endpoints return 100 items per page. The response includes `total` (full dataset size) and a `nextCursor` (`null` on the last page). Fetch the next page with `?before=<nextCursor>`. Ordering: `/sessions` by `server_ts DESC`, `/sessions/:id` by client `timestamp DESC`.

## Tests

Backend tests run in a throwaway Go container - no local Go install required:

```bash
docker run --rm -v "$(pwd)/backend:/app" -w //app golang:1.22-alpine go test ./...
```



The `//app` in the bash variant is intentional - it stops Git Bash's MSYS layer from rewriting `/app` to `C:/Program Files/Git/app`. Linux kernels normalize `//app` to `/app`, so it's safe everywhere bash runs.

## Design notes

- **Storage**: SQLite + WAL. `data/tracker.db` mounted as a Docker volume. Schema embedded via `go:embed`.
- **Ingest**: each `POST /events` is one SQLite transaction (all-or-nothing per batch). No server-side queue - the widget's in-memory queue (capped at 500, oldest dropped) is the system's only buffer.
- **Idempotency**: client-generated `eventId` (UUID) + `UNIQUE` index + `INSERT OR IGNORE`. Retries are safe.
- **Timestamps**: client `timestamp` is recorded but never authoritative for time windows. `server_ts` is set on insert and drives the "last 5 minutes" filter on `/sessions` and `/stats` - a wrong client clock cannot fabricate or hide recent activity.
- **Latency**: ~5 s widget batch + ~5 s dashboard poll = up to ~10 s end-to-end visibility.
- **Concurrency**: 25-conn pool. WAL gives concurrent reads alongside a single writer; `busy_timeout=5000` handles writer races.

### Scale

Throughput is gated by **write transactions/second**, not raw events. One `POST /events` = one transaction regardless of batch size.

| 1,000 sessions, 5s flush | Active sessions | Write TPS | Status |
|---|---|---|---|
| 8–10 % (typical content site) | ~80–100 | ~16–20 | Safe |
| 50 % (interactive) | ~500 | ~100 | At ceiling |
| 100 % | 1,000 | ~200 | Exceeds SQLite WAL |

Practical SQLite WAL ceiling is ~100–150 sustained TPS. **Migrate to Postgres** when sustained writes approach that, multiple backend instances are needed, or events table grows past ~10 GB. **Add Kafka** when writes exceed ~50k/s, multiple independent consumers are needed, or replay is required. The API contract and `schema.sql` shape don't change across either migration.
