# Design System — islami.click (v2)

> Acuan tunggal desain yang **sudah terimplementasi** di homepage redesign branch.
> Sumber: redesign hero (3c8a056), tiered features + active nav + logo (dc9096e).
> `docs/ref/` berisi referensi lokal (NIKE-DESIGN.md) — bukan sumber kebenaran.

## 1. Brand & Atmosphere

Quiet utility: ruang ibadah digital yang tenang, hangat, dan teratur. Deep teal (Iznik), warm gold (manuscript accent), ivory canvas (parchment). Arabic text IS the decoration — no geometric patterns, no calligraphy wallpaper.

## 2. Palette

| Role | Light | Dark | Notes |
|---|---|---|---|
| Canvas | `#FAF7F2` | `#0C1E26` | body gradient → `#EDE7DD` / `#08161C` |
| Card | `#FFFFFF` | `#122E38` | borders `#E2D9CE` / `#1E4458` |
| Primary teal | `#0E5C73` | `#83C0CC` (on dark) / `#A9D7DE` | hover `#0A4A5C`, deep `#083D4D`, substrate `#E6F4F8` / `#123B47` |
| Gold accent | `#C9A84C` | `#E4C76A` / `#806B35` | badges, dots, step numbers — **never primary actions**; surface `#F8F2E8` / `#2A2A1C`; text `#80631B` / `#C9A84C` |
| Title/Latin text | `#1A2E35` | `#FFFFFF`-warm (`#F5EFE4`) | |
| Body text | `#6F6256` / `#766A5E` / `#8A7E72` | `#B7C9CC` / `#AFC5C9` / `#7AAAB5` | tiered by emphasis |
| Dark elevated | — | `#143444`, `#102D37`, `#162F38`, `#0C2A35` | nav pill, hover, accent cards |
| Error | `#C0392B` | `#3D1008` | |
| Warm neutrals | `#F1ECE4`, `#EDE7DD`, `#D8CEC2`, `#DCCB9D` | `#315B68`, `#4A9BAD` | never stone/zinc/slate except legacy, no `#fff`/`#000` surfaces |

**No ALL-CAPS labels** — sentence case; micro-labels 13px bold tracking-wide (12px only when tiny meta).

## 3. Typography

- Latin: Plus Jakarta Sans (400/500/600/700), self-hosted `static/fonts/`
- Arabic: Amiri (400/700), self-hosted; `dir="rtl"`; min 28px, line-height 2.0+; feature-card Arabic `text-[#6F6256] dark:text-[#B7C9CC]`
- Hierarchy: H1 hero 36→60px bold tight; section H2 24–30px; card H3 base (16px) bold; body 14–15px `leading-relaxed`; countdown & prayer times always `tabular-nums`

## 4. Components

**Logo mark (Rub el Hizb refined)** — dua kotak rounded rx2.8 (23×23 pada viewBox48) `#FAF7F2`, rotate 45°, titik tengah `#C9A84C`, tile gradien `#0E5C73→#083D4D` (rounded-2xl, header h-12/sm:h-14, footer h-10). Wordmark: `islami`+`.click` teal (dark `#83C0CC`).

**Header nav (desktop pill)**: container `rounded-2xl border bg-[#F1ECE4]/70 dark:bg-[#102D37]/80 p-1`; item `min-h-11 px-3.5 rounded-xl text-sm font-medium`;
- idle: `text-[#5A4C40] hover:bg-white/80 hover:text-[#0E5C73]` (dark: `#D6C9B8` / `#143444` / `#A9D7DE`)
- **active: `bg-white text-[#0E5C73] shadow-sm` (dark: `bg-[#143444] text-[#A9D7DE]`)** — prefix-match utk `/quran/*`, `/quiz/*`
- Drawer mobile: active `bg-[#0E5C73]/10 text-[#0E5C73] dark:bg-[#143444] dark:text-[#A9D7DE]`

**Buttons**: `min-h-12 rounded-xl px-5 py-3 font-semibold`; primary `bg-[#0E5C73] text-white hover:bg-[#0A4A5C]`; secondary `border border-[#D8CEC2] bg-white/50 text-[#5A4C40] hover:border-[#0E5C73]` (dark `#315B68/#102D37/#D6E3E4`); `tap-press` feedback; focus-visible ring 2px teal.

**Homepage hero (2-col, `lg:grid-cols-[1.05fr_0.95fr] items-center`)**:
- Kiri: date line `Masehi · Hijri` (masehi muted, hijri semibold, gold dot divider) → H1 "Ibadahmu, lebih mudah." → sub (max-w-xl) → CTA `Mulai Dzikir` (primary, → /almatsurat) + `Baca Tafsir` (secondary, → /tafsir)
- Kanan (`space-y-4`): **live strip** — gold/teal `animate-ping` dot + `Sekarang HH:MM:SS WIB` + `{next} dalam [chip rounded-full bg-[#F1ECE4] font-semibold]` (mobile centered); **Next-prayer showcase card** — `Waktu Shalat {City}` + `Lihat semua →`, big `{Nama} {Jam}` (2xl/4xl bold tabular, name `#1A2E35`, time teal), 4 remaining prayer pills `bg-[#F1ECE4] rounded-xl py-2.5` (past `opacity-40`); card `rounded-[1.75rem] border bg-white p-5 sm:p-6`

**Feature grid (tiered & data-driven)** — data di `internal/model/home.go` `HomeFeatures{URL,Title,Desc,Arabic,Icon,Featured}`:
- **Featured (4)**: kicker `Jelajahi islami.click` → H2 `Teman ibadah harianmu`; kartu besar `rounded-[1.5rem] p-5 sm:p-6` — icon 12×12 teal substrate, Arabic 28px (min-h-[5.5rem]), H3 title, desc, `Buka →` (arrow-slide); hover `-translate-y-1 border-[#0E5C73]`
- **Fitur lainnya (compact tiles)**: label `text-xs font-semibold` → grid 2-col mobile / 4 lg; tile `rounded-2xl p-4` icon 11×11 + title (truncate) + desc (truncate text-xs) + chevron; hover `-translate-y-0.5`

**Shalat mini (htmx `/shalat/mini`)**: live values via Alpine `x-data` (update tiap detik); error state graceful; skeleton `h-52 animate-pulse` saat load.

## 5. Layout

- Container `max-w-6xl` (page nav & homepage), `max-w-3xl` reading pages; padding `px-4 sm:px-6`
- Hero `py-12 sm:py-16 lg:py-24`; sections `pb-20 sm:pb-24`
- Star/radius: buttons&inputs 12px (`rounded-xl`), cards 16px (`rounded-2xl`), hero cards 24–28px (`rounded-[1.5rem]/[1.75rem]`), pills `rounded-full`
- Elevation: borders define, shadows signal — `shadow-[0_16px_36px_-30px_rgba(35,55,61,0.65)]` cards, teal-tinted hover shadow; depth via color not cool greys
- Touch: min 44px (min-h-11/12); counter/tap zone full-width 64px+
- Dark mode: class-based `.dark` + OS preference on first load + manual toggle (localStorage `theme`); cache-bust CSS `out.css?v=N` — bump setiap regen

## 6. Do's & Don'ts (maintained rules)

- Teal = satu warna aksi; gold hanya aksen; tidak ada emerald/blue; no cool greys
- Arabic ≥28px, line-height 2.0+, RTL, *tidak pernah* auto-generate/guess
- Sentence case untuk label (no ALL-CAPS)
- No JS framework baru, no npm — Tailwind standalone, htmx 2, Alpine vendored
- Loading state hanya untuk htmx partial (skeleton); konten server-rendered
- Feature baru masuk `HomeFeatures` (bukan copy-paste HTML), `Featured: true` hanya untuk 4 core
