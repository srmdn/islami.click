# islami.click

Islamic content hub for Indonesian Muslims. Al-Ma'tsurat daily adhkar, du'a collection, and prayer times.

## Stack

Go + html/template for server-side rendering. htmx for partial updates. Alpine.js for client-side reactivity (tap counters, modals). Tailwind CSS v4 via standalone CLI binary — no Node, no npm, no build pipeline. SQLite via `modernc.org/sqlite` for content storage. Deploy target: Ubuntu 24.04 VPS, nginx, systemd.

## Run locally

```bash
# Terminal 1 — compile CSS
./tailwindcss -i static/css/input.css -o static/css/out.css --watch

# Terminal 2 — dev server
go run ./cmd/server
# http://localhost:8080
```

Build for production:

```bash
go build -o islami.click ./cmd/server
```

## Features

**`/`** — Landing page with Bismillah hero, 3 feature cards, prayer times widget.

**`/almatsurat`** — Wazifah Sugro and Kubro with tap-to-count per dhikr and visual progress bars. Progress resets on page reload.

**`/doa`** — 23 curated du'a across 7 categories plus ayat ruqyah. Source filter (Al-Qur'an / Hadits), category filter, full-text search, accordion, and load-more pagination.

**`/shalat`** — Prayer times with SQLite caching. Serves from cache after first daily fetch per city; falls back to stale cache if Aladhan API is down. Method=20 (Kemenag Indonesia), city picker, Hijri date, next-prayer highlight, mini widget for homepage. ±1–3 min variance from official Kemenag schedules.

## Project layout

```
cmd/server/main.go       entrypoint, router
internal/handler/         HTTP handlers per feature
internal/model/           domain types (dhikr, doa, shalat)
internal/store/           SQLite queries
migrations/               SQL migration files
templates/layouts/        base HTML layout
templates/pages/          per-page templates
templates/partials/       shared fragments (header, footer, shalat-mini, doa-more)
static/css/               Tailwind input + compiled output
static/js/                vendored htmx, Alpine.js
static/fonts/             self-hosted Arabic fonts (Amiri)
static/favicon.svg        SVG favicon (Rub el Hizb star)
content/                  JSON data (almatsurat, doa-harian, ayat-doa-ruqyah)
deploy/                   nginx + systemd configs
```

## Content rules

Arabic text is never auto-generated. All adhkar, du'a, and Quranic content must be verified against a primary source (mushaf or known printed edition) before committing.

## What's not here

No React, no Vue, no Vite, no Webpack. No Docker. No managed hosting. The Go binary is compiled and deployed directly to a VPS behind nginx.