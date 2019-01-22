package main

import (
	"fmt"
	"time"
)

const (
	englishUS = iota
	englishGB
	german
	langCount
)

var langNames = []string{
	englishUS: "English US",
	englishGB: "English GB",
	german:    "Deutsch",
}

func formatDate(t time.Time, lang int) string {
	y, m, d := t.Date()
	shortDay := shortDays[lang][t.Weekday()]
	switch lang {
	default: // English US
		return fmt.Sprintf("%s %02d/%02d/%04d", shortDay, m, d, y)
	case englishGB:
		return fmt.Sprintf("%s %02d/%02d/%04d", shortDay, d, m, y)
	case german:
		return fmt.Sprintf("%s %02d.%02d.%04d", shortDay, d, m, y)
	}
}
