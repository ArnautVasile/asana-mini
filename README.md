# asana-mini

Small Go utility that pulls **Workspaces**, **Users**, and **Projects** from Asana and writes them to JSON. Supports automatic retries on rate limits (`429` + `Retry-After`) and periodic sync on a fixed interval.

---

## Requirements

- Go 1.21+
- An Asana **Personal Access Token** (PAT)
- Check go installation by ```go version```

---

## Configuration

Create a `.env` file next to your working directory and add:

```
ASANA_PAT=your_asana_personal_access_token
# Optional:
# OUT_DIR=out
```

Environment variables override `.env` if both are set.

---

## Run

Choose one interval flag. The program runs immediately, then repeats on the chosen cadence, and stops cleanly on `Ctrl+C`.

```bash
go mod tidy

# 30-second interval
go run ./cmd/asana-mini --short_interval

# 5-minute interval
go run ./cmd/asana-mini --long_interval
```

If you use a `.env` file, you can omit `ASANA_PAT=...` in the command.

The app writes JSON per workspace into the configured output directory (`OUT_DIR`, default `out`).

---

## What it does

- Loads config from environment or `.env`.
- Calls Asana REST endpoints for workspaces, users, and projects.
- Follows pagination via `next_page.offset`.
- Handles `429 Too Many Requests` using `Retry-After`.
- Emits pretty-printed JSON files per workspace.

---

## Testing

Unit tests stub HTTP at the transport layer; no network calls are made.

```bash
go test ./...
```

---

## Notes

- Requires Asana scopes that allow reading workspaces, users, and projects associated with your token.  
- To change the output directory, set `OUT_DIR`.
