# islami.click

Islamic content hub for Indonesian Muslims.

## Stack

Go + html/template for server-side rendering. htmx for partial updates. Alpine.js for client-side reactivity. Tailwind CSS v4 via standalone CLI binary — no Node, no npm, no build pipeline. SQLite via `modernc.org/sqlite` for content storage.

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

**`/`** — Landing page with headline, daily Hijri date, prayer-times widget, and feature cards.

**`/almatsurat`** — Wazifah Sugro and Kubro with tap-to-count per dhikr and visual progress bars. Progress resets on page reload. Adhkar sourced from Al-Ma'tsurat by [Hasan Al-Banna](https://id.wikipedia.org/wiki/Hasan_al-Banna).

**`/doa`** — 23 curated du'a across 9 categories plus ayat ruqyah. Source filter (Al-Qur'an / Hadits), category filter, full-text search, accordion, and load-more pagination.

**`/shalat`** — Prayer times with SQLite caching. Serves from cache after first daily fetch per city; falls back to stale cache if Aladhan API is down. Method=20 (Kemenag Indonesia), city picker, Hijri date, next-prayer highlight, mini widget for homepage. ±1–3 min variance from official Kemenag schedules.

**`/asmaul-husna`** — 99 Names of Allah with Arabic, transliteration, and meaning.

**`/kiblat`** — Qibla direction compass using device geolocation.

**`/hisab`** — Hijri ↔ Masehi date converter with full calendar grid. Bidirectional conversion, important Islamic dates (Tahun Baru Islam, Asyura, Awal Ramadhan, Idul Fitri, Hari Arafah, Idul Adha), next Islamic event display, Hijriyah/Masehi month toggle, and Jumat (Friday) highlight.

**`/quran`** — Quran reader with per-surah browsing, Madinah mushaf pagination, smart search, audio recitation, and inline tafsir peek per ayah.

**`/tafsir`** — Tafsir reader with two editions (Al-Muyassar in Arabic, abridged Ibn Kathir in English), per-edition coverage, full-text search, and shareable per-ayah deep links.

**`/quiz`** — Quiz Islami: 8 categories (Aqidah, Rukun Islam, Al-Qur'an, Hadits, Sirah, Fiqh, Sejarah Islam, Akhlak), 3 difficulty levels (Basic 10 q, Intermediate/Advanced 15 q each), 30-second timer per question, time-bonus scoring (10 pts correct + up to 10 pts speed bonus), answer explanations, and a shared SQLite leaderboard per category and difficulty.

## Tafsir detail

**Multi-edition reader** — `/tafsir` lists all 114 surahs with per-edition commentary coverage. `/tafsir/:surah` renders verse Arabic, Indonesian translation, and commentary side by side with mushaf pagination. Edition switcher (`?edition=`), share-per-ayah deep links, and mushaf-style inline ayah markers.

**Search** (`/tafsir/search`) — Same reference grammar as Quran search (`2:255`, surah names); matches commentary text plus verse Arabic and Indonesian translation. Query-centered excerpts with keyword highlight.

**Inline peek** — Every ayah on `/quran/:surah` has a "Baca tafsir" control that expands the commentary inline via htmx (a plain deep-link without JS). The peek card offers an edition switcher and links out to the full reader.

## Quran detail

**Per-surah browsing** — `/quran` lists all 114 surahs with Arabic name, revelation type (Makkiyah/Madaniyah), and ayah count. `/quran/:surah` renders the surah with Arabic text (Madinah mushaf) and Indonesian translation.

**Mushaf pagination** — Ayahs are paginated by real Madinah mushaf page numbers, not arbitrary chunk sizes. Quran text and translation sourced from the quran-json dataset. htmx "Muat ayat berikutnya" loads the next mushaf page inline.

**Smart search** (`/quran/search`) — Four search strategies: direct references (`5:7`, `QS 36:1`), natural language (`ayat 7 al maidah`, `surah al baqarah ayat 255`), surah name lookup (`ar rahman`, `yasin`), and content search (`الحمد لله`, `segumpal darah`). Surah name normalization handles hyphens, apostrophes, and Indonesian translations.

**Audio** — Per-surah MP3 recitation by Mishari Rashid Alafasy via quranicaudio.com CDN. HTML5 `<audio>` element with Alpine.js play/pause toggle.

## Project layout

```
cmd/server/main.go         entrypoint, router
internal/handler/          HTTP handlers per feature
internal/model/            domain types (dhikr, doa, shalat, hisab, hijri, tafsir)
internal/store/            SQLite queries
internal/hijri/            Hijri ↔ Gregorian date conversion
migrations/               SQL migration files
templates/layouts/        base HTML layout
templates/pages/          per-page templates (incl. tafsir reader + search)
templates/partials/       shared fragments (header, footer, shalat-mini, doa-more, tafsir-peek)
static/css/               Tailwind input + compiled output
static/js/                vendored htmx, Alpine.js
static/fonts/             self-hosted Arabic fonts (Amiri)
static/favicon.svg        SVG favicon (Rub el Hizb star)
static/images/            OG/social preview image
content/                  JSON data (almatsurat, doa, quran, tafsir editions, prayer cities)
scripts/                  One-off utilities (fetch-quran, fetch-tafsir, fetch-prayer-cities)
deploy/                   nginx + systemd configs (placeholders — real values live on the VPS)
```

## Content rules

Arabic text is never auto-generated. All adhkar, du'a, and Quranic content must be verified against a primary source (mushaf or known printed edition) before committing.

## What's not here

No React, no Vue, no Vite, no Webpack. No Docker. No managed hosting.
