package utils

import (
	"fmt"
	"strconv"
	"time"
)

func GetCurrentTime() time.Time {
	return time.Now()
}

func GetFirstDayOfCurrentMonth() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
}

func GetCurrentYear() int {
	return time.Now().Year()
}

func FormatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// ExcelDateToTime converts Excel date (float64) to time.Time
func ExcelDateToTime(input interface{}) (string, error) {
	switch v := input.(type) {
	case string:
		// Try to parse as a string date first (e.g., "6/Jan/25")
		layouts := []string{
			"2/Jan/06",   // "6/Jan/25"
			"02/01/2006", // fallback for "06/01/2025"
			"1/2/2006",
			"2006-01-02",
			"2006/01/02",
		}

		for _, layout := range layouts {
			if t, err := time.Parse(layout, v); err == nil {
				return t.Format("2006-01-02"), nil
			}
		}

		// Try to parse string as Excel number (e.g., "31652")
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			d := excelSerialToTime(f)
			return d.Format("2006-01-02"), nil
		}

		return "", fmt.Errorf("unsupported string date format: %v", v)

	case float64:
		d := excelSerialToTime(v)
		return d.Format("2006-01-02"), nil

	default:
		return "nil", fmt.Errorf("unsupported Excel date format: %v", v)
	}
}

func excelSerialToTime(excelDate float64) *time.Time {
	// Excel's epoch is 1899-12-30 (yes, not 1900-01-01, due to historical bug)
	excelEpoch := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
	days := int64(excelDate)
	t := excelEpoch.AddDate(0, 0, int(days))
	return &t
}

// Formatted Date
func FormatDate(format string) string {
	timestamp := time.Now()
	if format == "" {
		return timestamp.Format("02/Jan/2006")
	}
	return timestamp.Format(format)
}

func FormatDate2(date *time.Time, format string) string {
	if date == nil {
		return ""
	}
	if format == "" {
		return date.Format("02/Jan/2006")
	}
	return date.Format(format)
}

func ParseDate(date *string) (*time.Time, error) {
	layout := "2006-01-02 15:04:05"

	if date == nil {
		return nil, nil
	}

	result, err := time.Parse(layout, *date)

	return &result, err
}

func ParseToSQLDate(date string) string {
	layout := "2006-01-02"

	if date == "" {
		return ""
	}

	result, _ := time.Parse(time.RFC3339Nano, date)

	return result.Format(layout)
}

func FormatDateStrToDate(date string) (string, error) {
	parsedDate, err := time.Parse(time.RFC3339, date)

	if err == nil {
		result := parsedDate.Format("2006-01-02")
		return result, nil
	}
	return "", err
}
