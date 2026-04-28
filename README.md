# islami.click

Islamic content hub for Indonesian Muslims.

## Stack

Go + html/template for server-side rendering. htmx for partial updates. Alpine.js for client-side reactivity. Tailwind CSS v4 via standalone CLI binary — no Node, no npm, no build pipeline. SQLite via `modernc.org/sqlite` for content storage. Deploy target: Ubuntu 24.04 VPS, nginx, systemd.

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

**`/`** — Landing page with Bismillah hero, feature cards, prayer times widget.

**`/almatsurat`** — Wazifah Sugro and Kubro with tap-to-count per dhikr and visual progress bars. Progress resets on page reload.

**`/doa`** — 23 curated du'a across 7 categories plus ayat ruqyah. Source filter (Al-Qur'an / Hadits), category filter, full-text search, accordion, and load-more pagination.

**`/shalat`** — Prayer times with SQLite caching. Serves from cache after first daily fetch per city; falls back to stale cache if Aladhan API is down. Method=20 (Kemenag Indonesia), city picker, Hijri date, next-prayer highlight, mini widget for homepage. ±1–3 min variance from official Kemenag schedules.

**`/asmaul-husna`** — 99 Names of Allah with Arabic, transliteration, and meaning.

**`/kiblat`** — Qibla direction compass using device geolocation.

**`/hisab`** — Hijri ↔ Masehi date converter with full calendar grid. Bidirectional conversion, important Islamic dates (Tahun Baru Islam, Asyura, Awal Ramadhan, Idul Fitri, Hari Arafah, Idul Adha), next event countdown, Hijriyah/Masehi month toggle, and Jumat (Friday) highlight.

**`/quran`** — Quran reader with per-surah browsing, Madinah mushaf pagination, smart search, and audio recitation.

## Quran detail

**Per-surah browsing** — `/quran` lists all 114 surahs with Arabic name, revelation type (Makkiyah/Madaniyah), and ayah count. `/quran/:surah` renders the surah with Arabic text (Madinah mushaf) and Indonesian translation (Kemenag).

**Mushaf pagination** — Ayahs are paginated by real Madinah mushaf page numbers, not arbitrary chunk sizes. Data sourced from quran.com API v4. htmx "Muat ayat berikutnya" loads the next mushaf page inline.

**Smart search** (`/quran/search`) — Four search strategies: direct references (`5:7`, `QS 36:1`), natural language (`ayat 7 al maidah`, `surah al baqarah ayat 255`), surah name lookup (`ar rahman`, `yasin`), and content search (`الحمد لله`, `segumpal darah`). Surah name normalization handles hyphens, apostrophes, and Indonesian translations.

**Audio** — Per-surah MP3 recitation by Mishari Rashid Alafasy via quranicaudio.com CDN. HTML5 `<audio>` element with Alpine.js play/pause toggle.

## Project layout

```
cmd/server/main.go         entrypoint, router
internal/handler/          HTTP handlers per feature
internal/model/            domain types (dhikr, doa, shalat, hisab, hijri)
internal/store/            SQLite queries
internal/hijri/            Hijri ↔ Gregorian date conversion
migrations/               SQL migration files
templates/layouts/        base HTML layout
templates/pages/          per-page templates
templates/partials/       shared fragments (header, footer, shalat-mini, doa-more)
static/css/               Tailwind input + compiled output
static/js/                vendored htmx, Alpine.js
static/fonts/             self-hosted Arabic fonts (Amiri)
static/favicon.svg        SVG favicon (Rub el Hizb star)
content/                  JSON data (almatsurat, doa-harian, ayat-doa-ruqyah, quran-surahs, quran-pages)
scripts/                  One-off utilities (fetch-quran, fetch-quran-pages)
deploy/                   nginx + systemd configs
```

## Content rules

Arabic text is never auto-generated. All adhkar, du'a, and Quranic content must be verified against a primary source (mushaf or known printed edition) before committing.

## What's not here

No React, no Vue, no Vite, no Webpack. No Docker. No managed hosting. The Go binary is compiled and deployed directly to a VPS behind nginx.
