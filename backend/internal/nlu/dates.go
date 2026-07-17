package nlu

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	jalaali "github.com/jalaali/go-jalaali"
)

// JalaliDate is a plain Jalali (Shamsi) calendar date.
type JalaliDate struct {
	Year  int
	Month int
	Day   int
}

// String returns the date as "YYYY-MM-DD", matching the departureDate
// query parameter 780.ir expects.
func (d JalaliDate) String() string {
	return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day)
}

func toJalaliDate(t time.Time) (JalaliDate, error) {
	jy, jm, jd, err := jalaali.ToJalaali(t.Year(), t.Month(), t.Day())
	if err != nil {
		return JalaliDate{}, err
	}
	return JalaliDate{Year: jy, Month: int(jm), Day: jd}, nil
}

func (d JalaliDate) toGregorian() (time.Time, error) {
	gy, gm, gd, err := jalaali.ToGregorian(d.Year, jalaali.Month(d.Month), d.Day)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(gy, gm, gd, 0, 0, 0, 0, time.UTC), nil
}

func addDaysJalali(now time.Time, days int) (JalaliDate, error) {
	return toJalaliDate(now.AddDate(0, 0, days))
}

var persianMonths = map[string]int{
	"فروردین": 1, "اردیبهشت": 2, "خرداد": 3, "تیر": 4, "مرداد": 5, "شهریور": 6,
	"مهر": 7, "آبان": 8, "آذر": 9, "دی": 10, "بهمن": 11, "اسفند": 12,
}

// weekdayNames uses space-separated forms (not ZWNJ) since ParseDate is
// expected to receive already-normalized text (Normalize turns ZWNJ into a
// space, e.g. "سه‌شنبه" -> "سه شنبه"). Ordered longest-name-first: "شنبه" is
// a substring of every other weekday name ("یکشنبه", "دوشنبه", ...), so it
// must be checked last or it would incorrectly match first.
var weekdayNames = []struct {
	name string
	day  time.Weekday
}{
	{"چهارشنبه", time.Wednesday},
	{"یک شنبه", time.Sunday},
	{"یکشنبه", time.Sunday},
	{"پنج شنبه", time.Thursday},
	{"پنجشنبه", time.Thursday},
	{"سه شنبه", time.Tuesday},
	{"سهشنبه", time.Tuesday},
	{"دوشنبه", time.Monday},
	{"جمعه", time.Friday},
	{"شنبه", time.Saturday},
}

var explicitDateRe = regexp.MustCompile(`(\d{1,2})\s*(فروردین|اردیبهشت|خرداد|تیر|مرداد|شهریور|مهر|آبان|آذر|دی|بهمن|اسفند)`)

// ParseDate extracts a Jalali date from normalized Persian text. It supports
// "امروز"/"فردا"/"پس‌فردا", weekday names (nearest future occurrence),
// "هفته بعد"/"هفته آینده", and explicit "DD MonthName" dates (assumes the
// current Jalali year unless that date has already passed, in which case
// next year is assumed). Returns nil if no date expression is found.
func ParseDate(text string, now time.Time) *JalaliDate {
	text = strings.TrimSpace(text)

	switch {
	case strings.Contains(text, "پس فردا"), strings.Contains(text, "پسفردا"):
		if d, err := addDaysJalali(now, 2); err == nil {
			return &d
		}
		return nil
	case strings.Contains(text, "امروز"):
		if d, err := toJalaliDate(now); err == nil {
			return &d
		}
		return nil
	case strings.Contains(text, "فردا"):
		if d, err := addDaysJalali(now, 1); err == nil {
			return &d
		}
		return nil
	}

	if strings.Contains(text, "هفته بعد") || strings.Contains(text, "هفته آینده") {
		if d, err := addDaysJalali(now, 7); err == nil {
			return &d
		}
		return nil
	}

	for _, wd := range weekdayNames {
		if strings.Contains(text, wd.name) {
			if d, err := nextWeekday(now, wd.day); err == nil {
				return &d
			}
			return nil
		}
	}

	return parseExplicitDate(text, now)
}

func nextWeekday(now time.Time, target time.Weekday) (JalaliDate, error) {
	diff := (int(target) - int(now.Weekday()) + 7) % 7
	if diff == 0 {
		diff = 7
	}
	return addDaysJalali(now, diff)
}

func parseExplicitDate(text string, now time.Time) *JalaliDate {
	m := explicitDateRe.FindStringSubmatch(text)
	if m == nil {
		return nil
	}
	day, err := strconv.Atoi(m[1])
	if err != nil || day < 1 || day > 31 {
		return nil
	}
	month, ok := persianMonths[m[2]]
	if !ok {
		return nil
	}

	todayJalali, err := toJalaliDate(now)
	if err != nil {
		return nil
	}

	candidate := JalaliDate{Year: todayJalali.Year, Month: month, Day: day}
	candidateGreg, err := candidate.toGregorian()
	if err != nil {
		return nil
	}

	nowDateOnly := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if candidateGreg.Before(nowDateOnly) {
		candidate.Year++
	}
	return &candidate
}
