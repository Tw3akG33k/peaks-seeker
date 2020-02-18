package data

import (
	"regexp"
	// "log"

	// "github.com/joho/godotenv"
	// "github.com/lucmichalski/peaks-seeker/env"
)

const (
	minScore = 20
)

var locationRegexp *regexp.Regexp

// ScoreOK checks whether the user has the minimum required score
func (u *User) ScoreOK() bool {
	return u.Score >= minScore
}

// DataOK checks whether the user has the minimum required fields matching
func (u *User) DataOK() bool {
	if !locationRegexp.MatchString(u.Location) {
		return false
	}

	if u.Email == "" {
		return false
	}

	return true
}

func SetLocations(loc string) {
	locationRegexp = regexp.MustCompile(`(?i)(` + loc + `)`)
}