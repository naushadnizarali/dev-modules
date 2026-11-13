package utils

import (
	"time"

	"go.uber.org/zap"
)

func AppLocation() *time.Location {
	name := GetEnv("APP_TIMEZONE", "UTC")
	if name == "" {
		name = "UTC"
	}

	location, err := time.LoadLocation(name)
	if err != nil {
		zap.L().Warn("invalid APP_TIMEZONE, using UTC", zap.String("timezone", name), zap.Error(err))
		return time.UTC
	}
	return location
}

func TimeNow() time.Time {
	return time.Now().In(AppLocation())
}
