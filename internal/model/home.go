package model

import "html/template"

type HomePageData struct {
	Meta        PageMeta
	HijriToday  string
	MasehiToday string
	Features    []HomeFeature
}

type HomeFeature struct {
	URL      string
	Title    string
	Desc     string
	Arabic   string
	Icon     template.HTML
	Featured bool
}

var HomeFeatures = []HomeFeature{
	{
		URL:      "/almatsurat",
		Title:    "Al-Ma'tsurat",
		Desc:     "Dzikir pagi dan petang dengan penghitung otomatis",
		Arabic:   "أَصْبَحْنَا وَأَصْبَحَ الْمُلْكُ لِلَّهِ",
		Icon:     template.HTML(`<circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/>`),
		Featured: true,
	},
	{
		URL:      "/doa",
		Title:    "Doa",
		Desc:     "Kumpulan doa harian lengkap dengan terjemah",
		Arabic:   "رَبَّنَا آتِنَا فِي الدُّنْيَا حَسَنَةً",
		Icon:     template.HTML(`<path d="M8 6h8M6 10h12M8 14h8"/><rect x="4" y="3" width="16" height="18" rx="3"/>`),
		Featured: true,
	},
	{
		URL:      "/quran",
		Title:    "Al-Qur'an",
		Desc:     "Baca Al-Qur'an lengkap dengan terjemahan",
		Arabic:   "قُلْ هُوَ اللَّهُ أَحَدٌ",
		Icon:     template.HTML(`<path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/>`),
		Featured: true,
	},
	{
		URL:      "/shalat",
		Title:    "Jadwal Shalat",
		Desc:     "Waktu shalat berdasarkan lokasi Anda",
		Arabic:   "إِنَّ الصَّلَاةَ كَانَتْ عَلَى الْمُؤْمِنِينَ كِتَابًا مَّوْقُوتًا",
		Icon:     template.HTML(`<path d="M3 12a9 9 0 1 0 18 0 9 9 0 0 0-18 0"/><path d="M12 2v2M12 20v2M2 12h2M20 12h2"/>`),
		Featured: true,
	},
	{
		URL:      "/tafsir",
		Title:    "Tafsir Al-Muyassar",
		Desc:     "Penjelasan ringkas tiap ayat Al-Qur'an",
		Arabic:   "كِتَابٌ أَنزَلْنَاهُ إِلَيْكَ مُبَارَكٌ",
		Icon:     template.HTML(`<path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/><line x1="9" y1="7" x2="15" y2="7"/><line x1="9" y1="11" x2="15" y2="11"/>`),
		Featured: false,
	},
	{
		URL:      "/asmaul-husna",
		Title:    "Asmaul Husna",
		Desc:     "99 Nama Allah dengan kaligrafi Arab dan makna",
		Arabic:   "ٱللَّهُ لَآ إِلَـٰهَ إِلَّا هُوَ ٱلْحَىُّ ٱلْقَيُّومُ",
		Icon:     template.HTML(`<path d="M12 2l3.09 6.26L22 9.27l-5 4.87L18.18 22 12 18.56 5.82 22 7 14.14l-5-4.87 6.91-1.01L12 2z"/>`),
		Featured: false,
	},
	{
		URL:      "/kiblat",
		Title:    "Arah Kiblat",
		Desc:     "Kompas arah Ka'bah berdasarkan lokasi",
		Arabic:   "فَوَلِّ وَجْهَكَ شَطْرَ الْمَسْجِدِ الْحَرَامِ",
		Icon:     template.HTML(`<circle cx="12" cy="12" r="10"/><polygon points="12,2 14,12 12,14 10,12" fill="currentColor" opacity="0.3" stroke="none"/><line x1="12" y1="2" x2="12" y2="14"/><circle cx="12" cy="12" r="2"/>`),
		Featured: false,
	},
	{
		URL:      "/hisab",
		Title:    "Hisab Hijriyah",
		Desc:     "Konversi tanggal Hijriyah dan Masehi",
		Arabic:   "إِنَّ عِدَّةَ الشُّهُورِ عِندَ ٱللَّهِ ٱثْنَا عَشَرَ شَهْرًا",
		Icon:     template.HTML(`<rect x="3" y="4" width="18" height="18" rx="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/>`),
		Featured: false,
	},
	{
		URL:      "/quiz",
		Title:    "Quiz Islami",
		Desc:     "Uji pengetahuan Islam dengan 8 kategori dan leaderboard",
		Arabic:   "هَلْ يَسْتَوِي الَّذِينَ يَعْلَمُونَ وَالَّذِينَ لَا يَعْلَمُونَ",
		Icon:     template.HTML(`<path d="M9 9a3 3 0 1 1 6 0c0 1.5-1.5 2-2.5 3"/><circle cx="12" cy="18" r="0.5" fill="currentColor"/><circle cx="12" cy="12" r="10"/>`),
		Featured: false,
	},
}
