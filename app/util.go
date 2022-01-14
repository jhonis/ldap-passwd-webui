package app

import (
	"os"
)

func getTitle() string {
	return envStr("LPW_TITLE", "Change your password on example.org")
}

func getPattern() string {
	return envStr("LPW_PATTERN", "^(?=.*[A-Z])(?=.*[!@#$&*])(?=.*\\d{1,})(?=.*[a-z]).{12,}$")
}

func getPatternInfo() string {
	return envStr("LPW_PATTERN_INFO", "Password must be at least 12 characters long\nPassword must contain an uppercase letter, a number and a special character\nDon't reuse any of your last 10 passwords")
}

func envStr(key, defaultValue string) string {
	val := os.Getenv(key)
	if val != "" {
		return val
	}
	return defaultValue
}
