package app

import (
	"os"
)

func getTitle() string {
	return envStr("LPW_TITLE", "Change your password on example.org")
}

func getPattern() string {
	return envStr("LPW_PATTERN", ".{8,}")
}

func getPatternInfo() string {
	return envStr("LPW_PATTERN_INFO", "Password must be at least 8 characters long.")
}

func envStr(key, defaultValue string) string {
	val := os.Getenv(key)
	if val != "" {
		return val
	}
	return defaultValue
}
