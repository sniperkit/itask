package format

import (
	"time"
)

func FormatDate(date time.Time) string {
	return date.Format(DateFormat)
}

func ParseDate(date string) (time.Time, error) {
	return time.Parse(DateFormat, date)
}
