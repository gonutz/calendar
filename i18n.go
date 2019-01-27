package main

const (
	englishUS = iota
	englishGB
	german
	langCount
)

var translations [langCount]translation

type translation struct {
	name        string
	windowTitle string
	menu        struct {
		today    string
		days     string
		weeks    string
		months   string
		language string
	}
	// calendarWeek is uesd in Sprintf with the integer week number as parameter
	calendarWeek string
	shortDays    [7]string
	// dateFormat will be called with Sprintf and the parameters are:
	// [1] <calendar week prefix>
	// [2] <day abbreviation>
	// [3] <day>
	// [4] <month>
	// [5] <year>
	dateFormat string
}

func init() {
	t := &translations[englishUS]
	t.name = "English US"
	t.windowTitle = "Calendar"
	t.menu.today = "&Today [F12]"
	t.menu.days = "&Days"
	t.menu.weeks = "&Weeks"
	t.menu.months = "&Months"
	t.menu.language = "&Language"
	t.calendarWeek = "CW %d  "
	t.shortDays = [7]string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"}
	t.dateFormat = "%[1]s%[2]s %02[4]d/%02[3]d/%04[5]d"

	translations[englishGB] = translations[englishUS]
	translations[englishGB].name = "English GB"
	translations[englishGB].dateFormat = "%s%s %02d/%02d/%04d"

	t = &translations[german]
	t.name = "Deutsch"
	t.windowTitle = "Kalender"
	t.menu.today = "&Heute [F12]"
	t.menu.days = "&Tage"
	t.menu.weeks = "&Wochen"
	t.menu.months = "&Monate"
	t.menu.language = "&Sprache"
	t.calendarWeek = "KW %d  "
	t.shortDays = [7]string{"So", "Mo", "Di", "Mi", "Do", "Fr", "Sa"}
	t.dateFormat = "%s%s %02d.%02d.%04d"
}
