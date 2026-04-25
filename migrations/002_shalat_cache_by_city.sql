DROP TABLE IF EXISTS prayer_time_cache;

CREATE TABLE prayer_time_cache (
    city TEXT NOT NULL,
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
    PRIMARY KEY (city, prayer_date, method)
);