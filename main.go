package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gonutz/w32"
	"github.com/gonutz/wui"
)

func main() {
	var settings appSettings
	settingsPath := filepath.Join(os.Getenv("APPDATA"), "calendar.set")
	if data, err := ioutil.ReadFile(settingsPath); err == nil {
		json.Unmarshal(data, &settings)
	}
	defer func() {
		data, err := json.Marshal(&settings)
		check(err)
		check(ioutil.WriteFile(settingsPath, data, 0666))
	}()

	font, _ := wui.NewFont(wui.FontDesc{Name: "Tahoma", Height: -13})
	bold, _ := wui.NewFont(wui.FontDesc{Name: "Tahoma", Height: -13, Bold: true})
	window := wui.NewWindow()
	window.SetFont(font)
	window.SetTitle("Calendar")
	window.SetIconFromExeResource(10)
	if settings.Maximized {
		window.Maximize()
	}
	window.SetShortcut(wui.ShortcutKeys{Key: w32.VK_ESCAPE}, window.Close)

	var editors [7 * 5]*editor
	for i := range editors {
		editors[i] = newEditor(window, font, bold)
	}
	hideEditors := func() {
		for _, e := range editors {
			e.setVisible(false)
		}
	}
	hideEditors()

	const (
		dayView = iota
		weekView
		monthView
		viewCount
	)

	if settings.View < 0 {
		settings.View = 0
	}
	settings.View %= viewCount
	if settings.Language < 0 || settings.Language >= langCount {
		settings.Language = englishUS
	}

	var cal calendar
	calPath := filepath.Join(os.Getenv("APPDATA"), "calendar")
	if data, err := ioutil.ReadFile(calPath); err == nil {
		json.Unmarshal(data, &cal)
	}
	defer func() {
		cal.clean()
		data, err := json.Marshal(&cal)
		check(err)
		check(ioutil.WriteFile(calPath, data, 0666))
	}()

	focus := time.Now()

	saveView := func() {
		for _, e := range editors {
			if e.visible() {
				cal.setText(e.date, fromWinLines(e.text()))
			}
		}
	}
	viewStart := func(view int, t time.Time) time.Time {
		switch view {
		case weekView:
			for t.Weekday() != time.Monday {
				t = t.Add(-24 * time.Hour)
			}
		case monthView:
			for t.Day() != 1 {
				t = t.Add(-24 * time.Hour)
			}
			for t.Weekday() != time.Monday {
				t = t.Add(-24 * time.Hour)
			}
		}
		return t
	}
	showDayView := func() {
		hideEditors()
		e := editors[0]
		e.setVisible(true)
		e.setBounds(0, 0, window.ClientWidth(), window.ClientHeight())
		e.setDate(focus, settings.Language)
		e.setText(toWinLines(cal.getText(e.date)))
		settings.View = dayView
	}
	showWeekView := func() {
		hideEditors()
		w := window.ClientWidth() / 7
		h := window.ClientHeight()
		hideEditors()
		offset := viewStart(weekView, focus)
		for i := 0; i < 7; i++ {
			e := editors[i]
			e.setVisible(true)
			width := w
			if i == 6 {
				width = window.ClientWidth() - 6*w
			}
			e.setBounds(i*w, 0, width, h)
			e.setDate(offset.Add(time.Duration(i)*24*time.Hour), settings.Language)
			e.setText(toWinLines(cal.getText(e.date)))
		}
		settings.View = weekView
	}
	showMonthView := func() {
		hideEditors()
		w := window.ClientWidth() / 7
		h := window.ClientHeight() / 5
		offset := viewStart(monthView, focus)
		for i := range editors {
			e := editors[i]
			e.setVisible(true)
			tx, ty := i%7, i/7
			width, height := w, h
			if tx == 6 {
				width = window.ClientWidth() - 6*w
			}
			if ty == 4 {
				height = window.ClientHeight() - 4*h
			}
			e.setBounds(tx*w, ty*h, width, height)
			e.setDate(offset.Add(time.Duration(i)*24*time.Hour), settings.Language)
			e.setText(toWinLines(cal.getText(e.date)))
		}
		settings.View = monthView
	}
	showView := func(view int, focusDelta time.Duration) {
		for _, e := range editors {
			if e.visible() && e.hasFocus() {
				focus = e.date
				break
			}
		}
		focus = focus.Add(focusDelta)
		saveView()
		switch view {
		case dayView:
			showDayView()
		case weekView:
			showWeekView()
		default:
			showMonthView()
		}
		for _, e := range editors {
			if e.visible() && e.date == focus {
				e.focus()
				break
			}
		}
	}

	nextView := func() {
		showView((settings.View+1)%viewCount, 0)
	}
	previousView := func() {
		showView((settings.View+viewCount-1)%viewCount, 0)
	}
	viewDelta := func(sign int) time.Duration {
		switch settings.View {
		case dayView:
			return 24 * time.Hour
		case weekView:
			return 7 * 24 * time.Hour
		default:
			return daysInMonth(focus, sign) * 24 * time.Hour
		}
	}
	moveForward := func() {
		showView(settings.View, viewDelta(1))
	}
	moveBackward := func() {
		showView(settings.View, -viewDelta(-1))
	}

	window.SetOnShow(func() {
		showView(settings.View, 0)
	})
	window.SetOnResize(func() {
		showView(settings.View, 0)
	})
	window.SetOnClose(func() {
		saveView()
		settings.Maximized = window.Maximized()
	})
	window.SetShortcut(wui.ShortcutKeys{Key: w32.VK_TAB}, func() {
		nextView()
	})
	window.SetShortcut(wui.ShortcutKeys{Key: w32.VK_TAB, Mod: wui.ModShift}, func() {
		previousView()
	})
	window.SetShortcut(wui.ShortcutKeys{Key: w32.VK_F1}, func() {
		moveBackward()
	})
	window.SetShortcut(wui.ShortcutKeys{Key: w32.VK_F2}, func() {
		moveForward()
	})

	menu := wui.NewMainMenu()
	window.SetMenu(menu)

	daysMenu := wui.NewMenuString("&Days")
	daysMenu.SetOnClick(func() {
		showView(dayView, 0)
	})
	menu.Add(daysMenu)

	weeksMenu := wui.NewMenuString("&Weeks")
	weeksMenu.SetOnClick(func() {
		showView(weekView, 0)
	})
	menu.Add(weeksMenu)

	monthsMenu := wui.NewMenuString("&Months")
	monthsMenu.SetOnClick(func() {
		showView(monthView, 0)
	})
	menu.Add(monthsMenu)

	backwardMenu := wui.NewMenuString("[F&1] <")
	backwardMenu.SetOnClick(moveBackward)
	menu.Add(backwardMenu)

	forwardMenu := wui.NewMenuString("> [F&2]")
	forwardMenu.SetOnClick(moveForward)
	menu.Add(forwardMenu)

	langMenu := wui.NewMenu("&Language")
	for lang := 0; lang < langCount; lang++ {
		m := wui.NewMenuString(langNames[lang])
		lang := lang
		m.SetOnClick(func() {
			settings.Language = lang
			showView(settings.View, 0)
		})
		langMenu.Add(m)
	}
	menu.Add(langMenu)

	window.Show()
}

type appSettings struct {
	Maximized bool
	View      int
	Language  int
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

type calendar struct {
	Dates []date
}

func (c *calendar) getText(t time.Time) string {
	y, m, d := t.Date()
	for i := range c.Dates {
		if d == c.Dates[i].Day &&
			int(m) == c.Dates[i].Month &&
			y == c.Dates[i].Year {
			return c.Dates[i].Text
		}
	}
	return ""
}

func (c *calendar) setText(t time.Time, text string) {
	y, m, d := t.Date()
	for i := range c.Dates {
		if d == c.Dates[i].Day &&
			int(m) == c.Dates[i].Month &&
			y == c.Dates[i].Year {
			c.Dates[i].Text = text
			return
		}
	}
	c.Dates = append(c.Dates, date{
		Day:     d,
		Month:   int(m),
		Year:    y,
		Weekday: int(t.Weekday()+6) % 7, // from [Su,Mo,...] to [Mo,Tu,...]
		Text:    text,
	})
}

func (c *calendar) clean() {
	nonEmpty := make([]date, 0, len(c.Dates))
	for i := range c.Dates {
		if c.Dates[i].Text != "" {
			nonEmpty = append(nonEmpty, c.Dates[i])
		}
	}
	sort.Sort(byDate(nonEmpty))
	c.Dates = nonEmpty
}

type date struct {
	Day     int // Day starts at 1
	Month   int // Month starts at 1 for January
	Year    int
	Weekday int // Weekday starts with 0 for Monday
	Text    string
}

type byDate []date

func (x byDate) Len() int { return len(x) }

func (x byDate) Less(i, j int) bool {
	a := x[i].Day + x[i].Month*31 + x[i].Year*670
	b := x[j].Day + x[j].Month*31 + x[j].Year*670
	return a < b
}

func (x byDate) Swap(i, j int) { x[i], x[j] = x[j], x[i] }

type editor struct {
	panel      *wui.Panel
	caption    *wui.Label
	edit       *wui.TextEdit
	date       time.Time
	font, bold *wui.Font
}

func newEditor(parent *wui.Window, font, bold *wui.Font) *editor {
	p := wui.NewPanel()
	parent.Add(p)
	e := wui.NewTextEdit()
	p.Add(e)
	l := wui.NewLabel()
	l.SetCenterAlign()
	p.Add(l)
	return &editor{
		panel:   p,
		caption: l,
		edit:    e,
		font:    font,
		bold:    bold,
	}
}

var shortDays = [][]string{
	englishUS: []string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"},
	englishGB: []string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"},
	german:    []string{"So", "Mo", "Di", "Mi", "Do", "Fr", "Sa"},
}

func (e *editor) setDate(t time.Time, lang int) {
	e.date = t
	e.caption.SetText(formatDate(t, lang))
	if isToday(t) {
		e.caption.SetFont(e.bold)
	} else {
		e.caption.SetFont(e.font)
	}
}

func isToday(t time.Time) bool {
	now := time.Now()
	y1, m1, d1 := now.Date()
	y2, m2, d2 := t.Date()
	return d1 == d2 && m1 == m2 && y1 == y2
}

func (e *editor) setVisible(v bool) {
	e.panel.SetVisible(v)
}

func (e *editor) visible() bool {
	return e.panel.Visible()
}

func (e *editor) bisible() bool {
	return e.panel.Visible()
}

func (e *editor) setBounds(x, y, w, h int) {
	e.panel.SetBounds(x, y, w, h)
	e.edit.SetBounds(0, 20, w, h-20)
	e.caption.SetBounds(0, 0, w, 20)
}

func (e *editor) setText(s string) {
	e.edit.SetText(s)
}

func (e *editor) text() string {
	return e.edit.Text()
}

func (e *editor) focus() {
	e.edit.Focus()
}

func (e *editor) hasFocus() bool {
	return e.edit.HasFocus()
}

func toWinLines(s string) string {
	return strings.Replace(s, "\n", "\r\n", -1)
}

func fromWinLines(s string) string {
	return strings.Replace(s, "\r\n", "\n", -1)
}

func daysInMonth(t time.Time, sign int) time.Duration {
	y, m, _ := t.Date()
	m-- // time.Month starts at 1, we want to start at 0
	if sign < 0 {
		m = (m + 11) % 12
	}
	if m == 1 { // February
		isLeapYear := y%400 == 0 || (y%4 == 0 && y%100 != 0)
		if isLeapYear {
			return 29
		}
		return 28
	}
	return []time.Duration{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}[m]
}
