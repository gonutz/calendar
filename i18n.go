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

var weekPrefix = []string{
	englishUS: "CW %d  ",
	englishGB: "CW %d  ",
	german:    "KW %d  ",
}

func formatDate(t time.Time, lang int) string {
	y, m, d := t.Date()
	weekday := t.Weekday()
	shortDay := shortDays[lang][weekday]
	var week string
	if weekday == time.Monday {
		_, w := t.ISOWeek()
		week = fmt.Sprintf(weekPrefix[lang], w)
	}
	switch lang {
	default: // English US
		return fmt.Sprintf("%s%s %02d/%02d/%04d", week, shortDay, m, d, y)
	case englishGB:
		return fmt.Sprintf("%s%s %02d/%02d/%04d", week, shortDay, d, m, y)
	case german:
		return fmt.Sprintf("%s%s %02d.%02d.%04d", week, shortDay, d, m, y)
	}
}
