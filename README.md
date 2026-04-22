# islami.click

Islamic content hub. Phase 1 covers Al-Ma'tsurat (daily adhkar), du'a collection, and prayer times for Indonesian users.

## Stack

Go + html/template for server-side rendering. htmx for partial updates. Alpine.js for client-side reactivity (tap counters, localStorage state). Tailwind CSS via standalone CLI binary -- no Node, no npm, no build pipeline.

SQLite via `modernc.org/sqlite` for content storage (seeded from JSON on startup). Deploy target: Ubuntu 24.04, nginx, systemd.

## Run locally

```bash
# Terminal 1 -- compile CSS
./tailwindcss -i static/css/input.css -o static/css/out.css --watch

# Terminal 2 -- dev server
go run ./cmd/server
# http://localhost:8080
```

Build for production:

```bash
go build -o islami.click ./cmd/server
```

## Features (Phase 1)

**`/almatsurat`** -- Wazifah Sugro and Kubro with tap-to-count per dhikr and visual progress bars. Progress resets on page reload (no persistence by design).

**`/doa`** -- Du'a collection with 23 curated entries across 7 categories. Filter by source (Al-Qur'an / Hadits), filter by category, full-text search, accordion expand, and load-more pagination. Sourced from `content/doa-harian.json`.

**`/shalat`** -- Prayer times via Aladhan API (method=20, Kemenag Indonesia), city picker, Hijri date. Fetched server-side, no client API calls.

## Project layout

```
cmd/server/          entrypoint, router
internal/handler/    HTTP handlers per feature
internal/model/      domain types
internal/store/      SQLite queries
templates/layouts/   base HTML layout
templates/pages/     per-page templates
templates/partials/  shared fragments (header, footer)
static/css/          Tailwind input + compiled output
static/js/           vendored htmx, Alpine.js
static/fonts/        self-hosted Arabic fonts
content/             JSON data (almatsurat, doa-harian)
deploy/              nginx + systemd configs
migrations/          SQL migration files
```

## Content rules

Arabic text is never auto-generated. All adhkar, du'a, and Quranic content must be verified against a primary source (mushaf or a known printed edition) before committing.

## What's not here

No React, no Vue, no Vite, no Webpack. No Docker. No managed hosting. The Go binary is compiled and deployed directly to a VPS behind nginx.
