# ProfilePulse

A lightweight, open-source GitHub profile view counter. Add a single line of Markdown to your profile README and ProfilePulse returns an SVG badge that increments an atomic counter on every request.

> [!IMPORTANT]
> Profile view counts are **request counts**, not unique visitors. GitHub may proxy or cache images, so the number should not be interpreted as a count of unique people.

---

## Table of Contents

- [How It Was Made](#how-it-was-made)
- [How It Works](#how-it-works)
- [How to Use](#how-to-use)

---

## How It Was Made

ProfilePulse is written in Go and built to run in two modes without any change to the source code: as a standalone HTTP server for local development and self-hosting, and as a Vercel Go Framework application for serverless deployment.

The project depends on a single external library — `github.com/redis/go-redis/v9` — for local Redis connectivity. The Upstash REST client is implemented from scratch using the standard `net/http` package, so no vendor SDK is required. The SVG badge renderer is also an original implementation: it builds SVG markup in memory from Go string templates, measures text width with a character-width heuristic, and escapes every dynamic value to prevent XML injection.

The codebase is organised into focused packages under `internal/`, each with a single responsibility:

| Package | Responsibility |
| --- | --- |
| `internal/badge` | SVG renderer with seven styles, number formatting, abbreviation, and XML escaping |
| `internal/config` | Environment variable loading and base-offset parsing |
| `internal/counter` | Store interface with Upstash REST, local Redis, and no-op implementations |
| `internal/server` | Shared HTTP handlers used by both the standalone server and Vercel |
| `internal/validation` | Input validation for username, colour, style, label, and base |

The `cmd/server/main.go` file wires these packages together into a standalone HTTP server. The `api/` directory contains thin entry points that delegate to the same handlers, kept for compatibility with Vercel's serverless function mode.

Every package carries its own test suite using Go's built-in `testing` package, with a mock counter store so tests never touch production Redis.

---

## How It Works

When a browser loads your GitHub profile, it requests the badge image from the `/api/badge` endpoint. The request flows through four stages:

1. **Validation** — The username is normalised, length-checked, and matched against a safe character set. Colour, style, label, and base parameters are validated against allow-lists.
2. **Counting** — A single atomic Redis `INCR` command increments the counter stored at `pv:v1:user:{username}` and returns the new value. No separate `GET` is issued, which conserves the Upstash free-tier command allowance.
3. **Formatting** — The raw count is combined with the optional base offset and, if requested, abbreviated to a compact form such as `1.2K`.
4. **Rendering** — The badge package generates the SVG markup, escaping all dynamic text, and the server returns it with `Content-Type: image/svg+xml` and cache-busting headers.

The store is selected automatically at startup. Upstash REST takes priority when its credentials are present; a local Redis URL is used next; otherwise a no-op store allows the service to start and serve health endpoints while returning a 503 for badge and count requests.

### Routes

| Method | Path | Response |
| --- | --- | --- |
| `GET` | `/` | `{"status":"ok"}` — JSON health payload |
| `GET` | `/health` | `{"status":"ok"}` — health check, no Redis required |
| `GET` | `/api/badge?username=...` | SVG badge image, counter incremented |
| `GET` | `/api/count?username=...` | JSON view count, counter incremented |

### Badge Styles

Seven visually distinct styles are available. The `bold` style is the default.

| Style | Height | Description |
| --- | --- | --- |
| `bold` | 28px | **(Default)** Large bold uppercase text, gradient segments, drop shadow, and a sweeping sheen animation |
| `flat` | 20px | Clean flat badge with rounded right corners and a subtle gradient overlay |
| `flat-square` | 20px | Same as `flat` with sharp square corners |
| `plastic` | 18px | Glossy badge with rounded corners and a gradient sheen |
| `capsule` | 22px | Fully rounded pill shape with flat colours |
| `outline` | 20px | Minimal badge with a 1px outer border and flat colours |
| `pixel` | 1×1 | Invisible counter image — the request is counted but no badge is shown |

> [!NOTE]
> The `pixel` style returns an invisible 1×1 image. The request is still counted, but no badge is displayed. The legacy `for-the-badge` style name is accepted as an alias for `bold` for backwards compatibility.

### Named Colours

`brightgreen`, `green`, `yellowgreen`, `yellow`, `orange`, `red`, `blue`, `grey`, `lightgrey`, `blueviolet`.

Bare 6-digit hex values without a leading `#` are also accepted.

---

## How to Use

### Add the Badge to Your Profile

Paste a single line of Markdown into your GitHub profile README:

```markdown
![Profile views](https://your-app.vercel.app/api/badge?username=YOUR_NAME)
```

### Customise the Badge

| Parameter | Default | Description |
| --- | --- | --- |
| `username` | *(required)* | GitHub username — alphanumeric and hyphens, max 39 characters |
| `color` | `blue` | Named colour or 6-digit hex without `#` |
| `style` | `bold` | One of the seven styles listed above |
| `label` | `Profile views` | Badge label text, max 64 characters |
| `base` | `0` | Display-only offset added to the stored count |
| `abbreviated` | `false` | Show `1.2K` instead of `1200` |

```markdown
![Profile views](https://your-app.vercel.app/api/badge?username=YOUR_NAME&color=red&style=flat-square&label=VIEWS&base=1000&abbreviated=true)
```

### Fetch the Count as JSON

```bash
curl "https://your-app.vercel.app/api/count?username=YOUR_NAME"
```

```json
{
  "username": "YOUR_NAME",
  "views": 1234,
  "raw": 1234,
  "display": "1234"
}
```

### Error Responses

| Status | Condition | Body |
| --- | --- | --- |
| `400` | Invalid `username`, `color`, `style`, `label`, or `base` | `{"error":"..."}` or error SVG badge |
| `503` | Redis unavailable | `{"error":"counter unavailable"}` or error SVG badge |

---

## Environment Variables

| Variable | Required | Description |
| --- | --- | --- |
| `UPSTASH_REDIS_REST_URL` | Yes (Vercel) | Upstash Redis REST URL |
| `UPSTASH_REDIS_REST_TOKEN` | Yes (Vercel) | Upstash Redis REST token |
| `REDIS_URL` | No (local) | Standard Redis URL for self-hosted or local development |
| `ADDR` | No | Server listen address (default `:3000`) |

> [!IMPORTANT]
> The store is selected automatically. Upstash REST takes priority, then local Redis, then a no-op store. Without any Redis backend, health endpoints still return `{"status":"ok"}` but badge and count endpoints return a 503 error.

See [`.env.example`](.env.example) for a template.

---

## Local Development

```bash
go mod download
go run ./cmd/server     # http://localhost:3000
```

> [!WARNING]
> You need a running Redis instance (or Upstash credentials) for the counter to work. Without Redis, the health endpoint still returns `{"status":"ok"}` but badge and count endpoints return a 503 error.

### Helper Scripts

| Script | Description |
| --- | --- |
| `scripts/test.sh` | Format check, vet, and tests with `-race` |
| `scripts/build.sh` | Build a stripped static binary to `bin/profilepulse` |
| `scripts/run.sh` | Run the dev server via `go run` |
| `scripts/lint.sh` | Run `golangci-lint` (if installed) |

---

## Testing

```bash
# Run all checks
./scripts/test.sh

# Or individually
gofmt -l .
go vet ./...
go test ./... -race -count=1
```

Tests use a mock counter store and do not require production Redis credentials. Coverage includes SVG rendering, escaping, number formatting, abbreviation, input validation, environment loading, base parsing, and HTTP handlers with edge cases.

GitHub Actions runs lint, test, and build on every push and pull request.

---

## Security & Privacy

- Strict input validation for username, colour, style, label, and base
- All dynamic text is XML-escaped before SVG generation
- No secrets in source; credentials come from environment variables
- Safe error messages with no stack traces or internal details leaked to clients
- `X-Content-Type-Options: nosniff` and `Referrer-Policy: no-referrer` headers
- Redis keys constructed from validated usernames only; no arbitrary key access
- Docker container runs as a non-root user

> [!IMPORTANT]
> No IP addresses, browser fingerprints, user agents, or personal visitor identities are stored. The counter stores only the aggregate count.
