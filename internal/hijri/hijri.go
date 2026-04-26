package hijri

import (
	"fmt"
	"time"
)

var MonthNamesID = [13]string{
	"", "Muharram", "Safar", "Rabiul Awal", "Rabiul Akhir",
	"Jumadil Awal", "Jumadil Akhir", "Rajab", "Sya'ban",
	"Ramadhan", "Syawal", "Dzulqa'dah", "Dzulhijjah",
}

var DayNamesID = [7]string{"Ahad", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}

var GregorianMonthNamesID = [13]string{
	"", "Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember",
}

type Date struct {
	Year  int
	Month int // 1-12
	Day   int // 1-30
}

func (h Date) ToGregorian() time.Time {
	jd := hijriToJDN(h.Year, h.Month, h.Day)
	y, m, d := jdnToGregorian(jd)
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
}

func (h Date) FormatID() string {
	g := h.ToGregorian()
	dayName := DayNamesID[g.Weekday()]
	monthName := MonthNamesID[h.Month]
	return dayName + ", " + itoa(h.Day) + " " + monthName + " " + itoa(h.Year) + " H"
}

func FromGregorian(t time.Time) Date {
	jd := gregorianToJDN(t.Year(), int(t.Month()), t.Day())
	y, m, d := jdnToHijri(jd)
	return Date{Year: y, Month: m, Day: d}
}

func FormatGregorianID(t time.Time) string {
	dayName := DayNamesID[t.Weekday()]
	monthName := GregorianMonthNamesID[t.Month()]
	return dayName + ", " + itoa(t.Day()) + " " + monthName + " " + itoa(t.Year())
}

func gregorianToJDN(y, m, d int) int {
	a := (14 - m) / 12
	y1 := y + 4800 - a
	m1 := m + 12*a - 3
	return d + (153*m1+2)/5 + 365*y1 + y1/4 - y1/100 + y1/400 - 32045
}

func hijriToJDN(y, m, d int) int {
	return (10631*y - 10617)/30 + (325*m - 320)/11 + d + 1948439
}

func jdnToGregorian(jd int) (int, int, int) {
	a := jd + 32044
	b := (4*a + 3) / 146097
	c := a - 146097*b/4
	d := (4*c + 3) / 1461
	e := c - (1461*d)/4
	m := (5*e + 2) / 153
	day := e - (153*m+2)/5 + 1
	month := m + 3 - 12*(m/10)
	year := 100*b + d - 4800 + m/10
	return year, month, day
}

func jdnToHijri(jd int) (int, int, int) {
	k2 := 30*(jd-1948440) + 15
	k1 := 11*((k2%10631)/30) + 5
	year := k2/10631 + 1
	month := k1/325 + 1
	day := (k1%325)/11 + 1
	return year, month, day
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}