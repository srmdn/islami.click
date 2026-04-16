# AGENTS.md — islami.click

Islam hub — Islamic content collection site.

## Stack

- Go + html/template + htmx + Alpine.js
- Tailwind CSS via standalone CLI (no Node/npm)
- SQLite via modernc.org/sqlite
- Deploy: Ubuntu 24.04 VPS, nginx, systemd, no Docker

## Project structure

```
assets.go                — Embed declarations (templates + content FS)
cmd/server/main.go       — Entrypoint, router, per-page template parsing
internal/handler/         — HTTP handlers per feature
internal/model/           — Domain types (dhikr, doa, user)
internal/store/           — SQLite queries
migrations/               — SQL migration files
templates/layouts/        — Base HTML layout (shared via {{define "base"}})
templates/pages/          — Page templates (define title/description/content blocks)
templates/partials/       — Reusable fragments (header, footer)
static/css/               — Tailwind input + compiled output
static/js/                — Vendored JS (htmx, Alpine.js)
static/fonts/             — Arabic fonts (self-hosted)
static/favicon.svg        — SVG favicon (Rub el Hizb / 8-pointed star)
content/                  — JSON data (almatsurat, doa, asmaul-husna)
deploy/                   — nginx + systemd configs
```

## Frontend design rules (for AI tools)

The frontend must look polished and modern, not like a bare admin panel.
Go templates produce HTML — the browser doesn't care what generated it.
Design quality comes from CSS + markup, not from the framework.

### Required frontend stack
- **Tailwind CSS** via standalone CLI binary (NOT npm). Download from GitHub releases.
- **htmx 2.x** vendored at `static/js/htmx.min.js` for server interactions
- **Alpine.js** vendored at `static/js/alpine.min.js` for client-side reactivity (counters, modals, transitions)
- **Google Fonts** for Arabic: Amiri or Scheherazade New (self-host in static/fonts/)

### Design standards
- Mobile-first, responsive
- Dark/light mode support via Tailwind `dark:` classes
- Arabic text: RTL, large line-height (2.0+), proper font
- Smooth transitions on state changes (htmx `hx-swap` + CSS transitions)
- Interactive elements must feel tactile (tap feedback, count animations)
- Use color intentionally: muted backgrounds, accent for progress/completion
- Cards, rounded corners, subtle shadows — not flat/brutalist
- Progress indicators must be visual (progress bars, circular counters), not just "3/10"

### What NOT to do
- Do not install Node, npm, Bun, or any JS build tool
- Do not add JS frameworks (React, Vue, Svelte, Preact)
- Do not use CSS-in-JS
- Do not over-engineer: htmx + Alpine.js covers all interactivity needed
- Do not use Tailwind CDN in production (use standalone CLI for purging)

## Content data shape (almatsurat)

Each adhkar entry in JSON:

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

## References

Study these before building:

- https://almatsurat.net/ — Main reference. Homepage with sugro/kubro navigation.
- https://almatsurat.net/sugro — Wazifah Sugro (short version). Key UX: tap-to-count per dhikr, progress display (0/3, 0/100), "Selesai" completion status, sequential flow from isti'adzah through doa robithoh.
- https://almatsurat.net/kubro — Wazifah Kubro (complete version). Same UX, longer content with more surah sections and higher repeat counts.

Replicate the core interaction: user taps an area to increment count toward target, visual progress updates, section marked done when target reached.

### Doa references
- https://quran.nu.or.id/doa — Main reference. Categorized doa collection (keseharian, shalat, wirid harian, etc). Each entry has Arabic text, Latin transliteration, and Indonesian translation.
- https://quran.nu.or.id/doa/doa-keseharian — Daily doa category. Study the content structure: doa title, Arabic, Latin, terjemah, source.
- https://quran.nu.or.id/doa/wirid-harian — Daily wirid. Similar structure to doa keseharian.

### Shalat (prayer times) references
- https://www.jadwalsholat.org/ — Main reference. Prayer times for all Indonesian cities, Gregorian-Hijri calendar, city selector, printable schedules.
- https://www.jadwalsholat.org/jadwal-sholat-hari-ini — Daily view. Study the layout: city picker, 5 prayer times display, hijri date.
- https://aladhan.com/prayer-times-api — API for prayer times data. Use method=20 (KMTT/Kemenag Indonesia) for Indonesian calculation.

## Phase 1 scope (current)

- `/almatsurat` — sugro + kubro with tap counter, progress tracking
- `/doa` — categorized du'a collection
- `/shalat` — prayer times via Aladhan API (fetched server-side)
- Counter state: localStorage via Alpine.js (no accounts yet)

## Phase 2 scope (later)

- User accounts, streak tracking, cross-device sync
- SQLite for persistent state
- Auth (session-based)

## Phase 3 scope (later)

- `/asmaul-husna`, `/kiblat`, `/hisab`, `/quran` reader
- Gamification (daily goals, badges, sharing)

## Hard rules

- Arabic text must be accurate — do not auto-generate or guess
- No npm/node/bun packages — ever
- No `.env`, `*.db`, `static/css/out.css` in commits
- Keep `CLAUDE.md` aligned with this file
- git config: `user.name "srmdn"`, `user.email "mail@saidwp.com"`
