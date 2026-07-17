package nlu

import "strings"

// Normalize cleans up raw Persian text (typically Web Speech API output)
// before any further analysis: it unifies Arabic/Persian letter variants,
// converts Persian and Arabic-Indic digits to Latin digits, turns ZWNJ
// (half-space) into a regular space, and collapses/trims whitespace.
func Normalize(text string) string {
	var b strings.Builder
	for _, r := range text {
		switch r {
		case 'ي':
			b.WriteRune('ی')
		case 'ك':
			b.WriteRune('ک')
		case '۰', '٠':
			b.WriteRune('0')
		case '۱', '١':
			b.WriteRune('1')
		case '۲', '٢':
			b.WriteRune('2')
		case '۳', '٣':
			b.WriteRune('3')
		case '۴', '٤':
			b.WriteRune('4')
		case '۵', '٥':
			b.WriteRune('5')
		case '۶', '٦':
			b.WriteRune('6')
		case '۷', '٧':
			b.WriteRune('7')
		case '۸', '٨':
			b.WriteRune('8')
		case '۹', '٩':
			b.WriteRune('9')
		case '‌': // ZWNJ (half-space)
			b.WriteRune(' ')
		default:
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
