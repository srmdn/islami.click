# CLAUDE.md — islami.click

Islamic content hub for Indonesian Muslims.

## Stack

- Go + html/template + htmx + Alpine.js
- Tailwind CSS v4 via standalone CLI (no Node/npm)
- SQLite via modernc.org/sqlite
- Deploy: Ubuntu 24.04 VPS, nginx, systemd, no Docker

## Commands

```bash
go run ./cmd/server                          # Dev server (port 8080)
./tailwindcss -i static/css/input.css -o static/css/out.css --watch  # CSS dev
go build -o islami.click ./cmd/server        # Production build
go test ./...                                # Run tests
```

## Project structure

```
cmd/server/main.go       — Entrypoint, router, per-page template parsing
internal/handler/          — HTTP handlers per feature
internal/model/           — Domain types (dhikr, doa, shalat)
internal/store/           — SQLite queries
migrations/               — SQL migration files
templates/layouts/        — Base HTML layout ({{define "base"}})
templates/pages/          — Page templates (title/description/content blocks)
templates/partials/       — Reusable fragments (header, footer, shalat-mini, doa-more)
static/css/               — Tailwind input + compiled output
static/js/                — Vendored JS (htmx, Alpine.js)
static/fonts/             — Arabic fonts (self-hosted Amiri)
static/favicon.svg        — SVG favicon (Rub el Hizb star)
content/                  — JSON data (almatsurat-sugro, almatsurat-kubro, doa-harian, ayat-doa-ruqyah)
deploy/                   — nginx + systemd configs
```

## Frontend design rules

Full design system: `docs/ref/lp/ISLAMICLICK-DESIGN.md`

Key rules:
- Warm ivory canvas (`#FAF7F2` light, `#0C1E26` dark), deep teal primary (`#0E5C73`), warm gold accent (`#C9A84C`)
- Arabic text is the visual anchor — Amiri font, RTL, line-height 2.0+, never compress
- Plus Jakarta Sans for all Latin text
- Mobile-first, responsive, full dark mode via `dark:` classes
- Cards with warm borders (`#E2D9CE` light, `#1E4458` dark), rounded corners, minimal shadow
- No cool greys (`slate-*`, `gray-*`, `zinc-*`) — always warm tones
- Interactive elements: 44px touch targets, tap feedback, smooth transitions
- No Islamic geometric patterns as decoration — Arabic text IS the decoration
- htmx 2.x for partial updates, Alpine.js for client reactivity

### Required frontend stack
- **Tailwind CSS v4** via standalone CLI binary (NOT npm)
- **htmx 2.x** vendored at `static/js/htmx.min.js`
- **Alpine.js** vendored at `static/js/alpine.min.js`
- **Amiri** font self-hosted at `static/fonts/`

### What NOT to do
- Do not install Node, npm, Bun, or any JS build tool
- Do not add JS frameworks (React, Vue, Svelte, Preact)
- Do not use CSS-in-JS
- Do not use cool grey palette (`slate`, `gray`, `zinc`) — use warm tones only
- Do not use Tailwind CDN in production

## Content data shape

### Almatsurat (dhikr)
```json
{
  "id": "isti-adzah",
  "type": "quran",
  "title": "Isti'adzah",
  "arabic": "...",
  "translation": "...",
  "repeat": 1,
  "source": "HR. Abu Dawud"
}
```

### Doa
```json
{
  "id": "doa-sebelum-makan",
  "category": "makanan",
  "source_type": "hadits",
  "title": "Doa Sebelum Makan",
  "arabic": "...",
  "translation": "...",
  "source": "HR. Abu Dawud"
}
```

## Phase 1 scope (complete)

- `/` — Landing page with Bismillah hero, feature cards, prayer times widget
- `/almatsurat` — Sugro + Kubro with tap counter, progress resets on reload
- `/doa` — 23 curated du'a across 7 categories, source filter, search, accordion, pagination, + ruqyah
- `/shalat` — Prayer times with SQLite caching (method=20 Kemenag), city picker, Hijri date, next-prayer highlight, mini widget

## Phase 2 scope (complete)

- Cache prayer times in SQLite — on-demand per city, expires daily, stale fallback on API failure
- Checksum-aware content seeding — starts up idempotently, only re-seeds changed JSON collections

## Phase 2 scope (next — content features)

- `/asmaul-husna` — 99 Names of Allah with Arabic, transliteration, meaning
- `/kiblat` — Qibla direction compass using device geolocation
- `/hisab` — Hijri calendar converter and Islamic date display
- `/quran` — Quran reader with per-surah browsing and audio

## Deferred

- User accounts, streak tracking, cross-device sync — useful for a platform, not a utility. `localStorage` suffices for progress tracking.
- Auth (session-based) — no login wall for an Islamic content site. Speed of access beats identity.

## Phase 3 scope (later)

- Gamification (daily goals, badges, sharing)

## Hard rules

- Arabic text must be accurate — do not auto-generate or guess
- No npm/node/bun packages — ever
- No `.env`, `*.db`, `static/css/out.css` in commits
- git config: `user.name "srmdn"`, `user.email "mail@saidwp.com"`
- No footer lines in commit messages — no "Ultraworked with", no "Co-authored-by: Sisyphus", no attribution trailers