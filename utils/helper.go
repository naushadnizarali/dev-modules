package utils

import (
	"encoding/json"
	"log"
	"strings"
	"unicode"
	"uuid"

	"golang.org/x/crypto/bcrypt"
)

var basePath = GetEnv("ENV", "development")

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

func NewUUID() string {
	id := uuid.NewV7()

	return id.String()
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

func ToJsonString(input interface{}) string {
	result, err := json.Marshal(input)
	if err != nil {
		return ""
	}
	return string(result)
}

func ExtractIDs[T any, ID comparable](items []T, getID func(T) ID) []ID {
	ids := make([]ID, 0, len(items))
	for _, item := range items {
		ids = append(ids, getID(item))
	}
	return ids
}
