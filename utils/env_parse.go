package utils

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
)

func GetEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func GetInt(key string, fallback int) int {
	v, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key)))
	if err != nil {
		return fallback
	}
	return v
}

func GetBool(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func SplitCSV(s string) []string {
	if s == "" || s == "*" {
		return []string{"*"}
	}
	var out []string
	for part := range strings.SplitSeq(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func ParseLevel(levelStr string) slog.Level {
	if levelStr == "" {
		return slog.LevelInfo
	}

	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		slog.Warn("invalid log level, defaulting to INFO", "level", levelStr)
		return slog.LevelInfo
	}
}
