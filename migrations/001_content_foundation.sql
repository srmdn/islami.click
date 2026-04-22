CREATE TABLE IF NOT EXISTS content_collections (
    id TEXT PRIMARY KEY,
    kind TEXT NOT NULL CHECK (kind IN ('doa', 'dhikr', 'asmaul_husna', 'quran')),
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    source_path TEXT NOT NULL,
    source_checksum TEXT NOT NULL,
    display_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS content_categories (
    id INTEGER PRIMARY KEY,
    collection_id TEXT NOT NULL REFERENCES content_collections(id) ON DELETE CASCADE,
    slug TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    display_order INTEGER NOT NULL DEFAULT 0,
    UNIQUE (collection_id, slug)
);

CREATE TABLE IF NOT EXISTS content_items (
    id INTEGER PRIMARY KEY,
    collection_id TEXT NOT NULL REFERENCES content_collections(id) ON DELETE CASCADE,
    slug TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL,
    arabic TEXT NOT NULL DEFAULT '',
    latin TEXT NOT NULL DEFAULT '',
    translation TEXT NOT NULL DEFAULT '',
    repeat_count INTEGER NOT NULL DEFAULT 1,
    source TEXT NOT NULL DEFAULT '',
    source_url TEXT NOT NULL DEFAULT '',
    verification TEXT NOT NULL DEFAULT '',
    metadata_json TEXT NOT NULL DEFAULT '{}',
    display_order INTEGER NOT NULL DEFAULT 0,
    UNIQUE (collection_id, slug)
);

CREATE TABLE IF NOT EXISTS content_item_categories (
    item_id INTEGER NOT NULL REFERENCES content_items(id) ON DELETE CASCADE,
    category_id INTEGER NOT NULL REFERENCES content_categories(id) ON DELETE CASCADE,
    display_order INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (item_id, category_id)
);

CREATE INDEX IF NOT EXISTS idx_content_categories_collection_order
    ON content_categories(collection_id, display_order);

CREATE INDEX IF NOT EXISTS idx_content_items_collection_order
    ON content_items(collection_id, display_order);

CREATE INDEX IF NOT EXISTS idx_content_item_categories_category_order
    ON content_item_categories(category_id, display_order);

CREATE TABLE IF NOT EXISTS asmaul_husna_names (
    number INTEGER PRIMARY KEY CHECK (number BETWEEN 1 AND 99),
    slug TEXT NOT NULL UNIQUE,
    arabic TEXT NOT NULL,
    latin TEXT NOT NULL,
    translation TEXT NOT NULL,
    explanation TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS quran_surahs (
    number INTEGER PRIMARY KEY CHECK (number BETWEEN 1 AND 114),
    slug TEXT NOT NULL UNIQUE,
    name_arabic TEXT NOT NULL,
    name_latin TEXT NOT NULL,
    name_translation TEXT NOT NULL DEFAULT '',
    revelation_place TEXT NOT NULL DEFAULT '',
    ayah_count INTEGER NOT NULL CHECK (ayah_count > 0)
);

CREATE TABLE IF NOT EXISTS quran_ayahs (
    surah_number INTEGER NOT NULL REFERENCES quran_surahs(number) ON DELETE CASCADE,
    ayah_number INTEGER NOT NULL CHECK (ayah_number > 0),
    text_arabic TEXT NOT NULL,
    translation TEXT NOT NULL DEFAULT '',
    tafsir TEXT NOT NULL DEFAULT '',
    juz INTEGER NOT NULL DEFAULT 0,
    page INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (surah_number, ayah_number)
);

CREATE TABLE IF NOT EXISTS locations (
    id INTEGER PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    label TEXT NOT NULL,
    city TEXT NOT NULL DEFAULT '',
    country TEXT NOT NULL DEFAULT 'Indonesia',
    latitude REAL,
    longitude REAL,
    timezone TEXT NOT NULL DEFAULT 'Asia/Jakarta',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS prayer_time_cache (
    location_id INTEGER NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
    prayer_date TEXT NOT NULL,
    method INTEGER NOT NULL DEFAULT 20,
    imsak TEXT NOT NULL DEFAULT '',
    fajr TEXT NOT NULL,
    sunrise TEXT NOT NULL,
    dhuhr TEXT NOT NULL,
    asr TEXT NOT NULL,
    maghrib TEXT NOT NULL,
    isha TEXT NOT NULL,
    hijri_date TEXT NOT NULL DEFAULT '',
    raw_json TEXT NOT NULL DEFAULT '{}',
    fetched_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TEXT NOT NULL,
    PRIMARY KEY (location_id, prayer_date, method)
);

CREATE TABLE IF NOT EXISTS qibla_cache (
    location_id INTEGER PRIMARY KEY REFERENCES locations(id) ON DELETE CASCADE,
    bearing_degrees REAL NOT NULL,
    distance_km REAL,
    calculated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
