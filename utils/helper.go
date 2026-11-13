package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

var basePath = os.Getenv("ENV")

// Hash the password with bcrypt
func HashPassword(password string) string {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		return ""
	}
	return string(bytes)
}

// ComparePassword compares a plain password with a hashed password
func ComparePassword(storedHashedPassword string, inputPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(storedHashedPassword), []byte(inputPassword))
	if err != nil {
		log.Printf("Error comparing passwords: %v", err)
		return false
	}
	return err == nil // If no error, the passwords match
}

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

func StringToUint(s string) uint {
	val, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return uint(val)
}

func StringToUintPtr(s string) *uint {
	num, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return nil
	}
	n := uint(num)
	return &n
}

func StringToUint32(s string) uint32 {
	num, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0
	}
	return uint32(num)
}

func StringToUint32Ptr(s string) *uint32 {
	num, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return nil
	}
	n := uint32(num)
	return &n
}

func StringToUint64(s string) uint64 {
	num, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return num
}

func GetInitials(str string) string {
	delimiter := " "
	if strings.Contains(str, " ") {
		delimiter = " "
	} else if strings.Contains(str, ".") {
		delimiter = "."
	}

	words := strings.Split(str, delimiter)
	initials := ""

	for _, word := range words {
		trimmed := strings.TrimSpace(word)
		if trimmed != "" {
			runes := []rune(trimmed)
			initials += string(unicode.ToUpper(runes[0]))
		}
	}

	return initials
}

func GetFilePath(path string) string {
	return basePath + "/" + path
}

func PtrString(s string) *string {
	return &s
}

func PtrUInt(i uint) *uint {
	return &i
}

func StringPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func UIntPtr(i *uint) uint {
	return *i
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

// DownloadFile downloads a file from a given URL and returns the local path
func DownloadFile(url string) (string, string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	// Extract fileName from URL
	fileName := filepath.Base(resp.Request.URL.Path)

	// Create a temp file with proper extension
	tmpFile, err := os.CreateTemp("", "email-att-*"+filepath.Ext(fileName))
	if err != nil {
		return "", "", err
	}
	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return "", "", err
	}

	return tmpFile.Name(), fileName, nil
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

func ToJsonString(input interface{}) string {
	result, err := json.Marshal(input)
	if err != nil {
		return ""
	}
	return string(result)
}

func FormatDateStrToDate(date string) (string, error) {
	parsedDate, err := time.Parse(time.RFC3339, date)

	if err == nil {
		result := parsedDate.Format("2006-01-02")
		return result, nil
	}
	return "", err
}

func ExtractIDs[T any, ID comparable](items []T, getID func(T) ID) []ID {
	ids := make([]ID, 0, len(items))
	for _, item := range items {
		ids = append(ids, getID(item))
	}
	return ids
}
