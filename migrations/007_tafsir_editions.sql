-- Multi-edition tafsir: one ayah can carry commentary from many books.
--
-- Before this migration, quran_ayahs.tafsir held exactly one edition
-- (At-Tafsir Al-Muyassar, Arabic). That column stays for backward
-- compatibility while handlers/templates still read it; new code reads
-- tafsir_texts filtered by edition_id instead.
--
-- The backfill below is idempotent (INSERT OR IGNORE). On a fresh
-- database it inserts nothing because seeding runs after migrations;
-- seedTafsir then fills both the legacy column and tafsir_texts.

CREATE TABLE IF NOT EXISTS tafsir_editions (
    id TEXT PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    author TEXT NOT NULL DEFAULT '',
    language TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL DEFAULT '',
    resource TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tafsir_texts (
    edition_id TEXT NOT NULL REFERENCES tafsir_editions(id) ON DELETE CASCADE,
    surah_number INTEGER NOT NULL,
    ayah_number INTEGER NOT NULL CHECK (ayah_number > 0),
    text TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (edition_id, surah_number, ayah_number)
);

CREATE INDEX IF NOT EXISTS idx_tafsir_texts_surah
    ON tafsir_texts(edition_id, surah_number);

INSERT OR IGNORE INTO tafsir_editions (id, slug, title, author, language, source, resource) VALUES
    ('muyassar', 'al-muyassar', 'Tafsir Al-Muyassar',
     'Sekelompok ulama di bawah Syaikh Dr. Shalih Alusy-Syaikh',
     'arabic', 'Quran.com API v4', 'ar-tafsir-muyassar (resource 16)'),
    ('ibn-kathir-en', 'ibn-kathir-en', 'Ibn Kathir (Abridged)',
     'Hafiz Ibn Kathir',
     'english', 'Quran.com API v4', 'en-tafisr-ibn-kathir (resource 169)');

INSERT OR IGNORE INTO tafsir_texts (edition_id, surah_number, ayah_number, text)
    SELECT 'muyassar', surah_number, ayah_number, tafsir
    FROM quran_ayahs
    WHERE tafsir <> '';
