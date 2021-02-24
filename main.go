package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/gonutz/w32/v2"
	"github.com/gonutz/wui/v2"
)

func main() {
	const (
		dayView = iota
		weekView
		monthView
		viewCount
	)

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
	if settings.View < 0 {
		settings.View = 0
	}
	settings.View %= viewCount
	if settings.Language < 0 || settings.Language >= langCount {
		settings.Language = englishUS
	}

	font, _ := wui.NewFont(wui.FontDesc{Name: "Tahoma", Height: -13})
	bold, _ := wui.NewFont(wui.FontDesc{Name: "Tahoma", Height: -13, Bold: true})
	window := wui.NewWindow()
	window.SetFont(font)
	window.SetTitle(translations[settings.Language].windowTitle)
	icon, _ := wui.NewIconFromExeResource(10)
	window.SetIcon(icon)
	if settings.Maximized {
		window.SetState(wui.WindowMaximized)
	}
	window.SetShortcut(window.Close, wui.KeyEscape)
	// The settings store the last top-left corner of the monitor which the
	// window was previously on. On the next program start this monitor might be
	// unplugged. Since we do not want to show the window on a non-existing
	// monitor (this would put the window off-screen) we check if there
	// currently is a monitor that has the same top-left corner and put our
	// window on it.
	var monitors []w32.HMONITOR
	cb := syscall.NewCallback(func(m w32.HMONITOR, hdc w32.HDC, r *w32.RECT, l w32.LPARAM) uintptr {
		monitors = append(monitors, m)
		return 1
	})
	w32.EnumDisplayMonitors(0, nil, cb, 0)
	for _, monitor := range monitors {
		var info w32.MONITORINFO
		if w32.GetMonitorInfo(monitor, &info) &&
			int(info.RcWork.Left) == settings.MonitorX &&
			int(info.RcWork.Top) == settings.MonitorY {
			window.SetPosition(settings.MonitorX, settings.MonitorY)
			break
		}
	}

	var editors [7 * 5]*editor
	for i := range editors {
		editors[i] = newEditor(window, font, bold)
	}
	for _, e := range editors {
		e.setVisible(false)
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
		for _, e := range editors[1:] {
			e.setVisible(false)
		}
		e := editors[0]
		e.setVisible(true)
		e.setBounds(0, 0, window.InnerWidth(), window.InnerHeight())
		e.setDate(focus, settings.Language)
		e.setText(toWinLines(cal.getText(e.date)))
		settings.View = dayView
	}
	showWeekView := func() {
		for _, e := range editors[7:] {
			e.setVisible(false)
		}
		w := window.InnerWidth() / 7
		h := window.InnerHeight()
		offset := viewStart(weekView, focus)
		for i := 0; i < 7; i++ {
			e := editors[i]
			e.setVisible(true)
			width := w
			if i == 6 {
				width = window.InnerWidth() - 6*w
			}
			e.setBounds(i*w, 0, width, h)
			e.setDate(offset.Add(time.Duration(i)*24*time.Hour), settings.Language)
			e.setText(toWinLines(cal.getText(e.date)))
		}
		settings.View = weekView
	}
	showMonthView := func() {
		w := window.InnerWidth() / 7
		h := window.InnerHeight() / 5
		offset := viewStart(monthView, focus)
		for i := range editors {
			e := editors[i]
			e.setVisible(true)
			tx, ty := i%7, i/7
			width, height := w, h
			if tx == 6 {
				width = window.InnerWidth() - 6*w
			}
			if ty == 4 {
				height = window.InnerHeight() - 4*h
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
	showToday := func() {
		// do this twice because the first will reset the focus
		showView(settings.View, 0)
		showView(settings.View, time.Now().Sub(focus))
	}

	window.SetOnShow(func() {
		showView(settings.View, 0)
	})
	window.SetOnResize(func() {
		showView(settings.View, 0)
	})
	window.SetOnClose(func() {
		saveView()
		settings.Maximized = window.State() == wui.WindowMaximized
		monitor := window.Monitor()
		if monitor != 0 {
			var info w32.MONITORINFO
			if w32.GetMonitorInfo(w32.HMONITOR(monitor), &info) {
				settings.MonitorX = int(info.RcWork.Left)
				settings.MonitorY = int(info.RcWork.Top)
			}
		}
	})
	window.SetShortcut(nextView, wui.KeyTab)
	window.SetShortcut(previousView, wui.KeyShift, wui.KeyTab)
	window.SetShortcut(moveBackward, wui.KeyF1)
	window.SetShortcut(moveForward, wui.KeyF2)
	window.SetShortcut(
		func() { showView(settings.View, -24*time.Hour) },
		wui.KeyAlt, wui.KeyLeft,
	)
	window.SetShortcut(
		func() { showView(settings.View, 24*time.Hour) },
		wui.KeyAlt, wui.KeyRight,
	)
	window.SetShortcut(
		func() {
			if settings.View == monthView {
				showView(settings.View, -7*24*time.Hour)
			}
		},
		wui.KeyAlt, wui.KeyUp,
	)
	window.SetShortcut(
		func() {
			if settings.View == monthView {
				showView(settings.View, 7*24*time.Hour)
			}
		},
		wui.KeyAlt, wui.KeyDown,
	)

	menu := wui.NewMainMenu()
	window.SetMenu(menu)

	todayMenu := wui.NewMenuString(translations[settings.Language].menu.today)
	todayMenu.SetOnClick(showToday)
	window.SetShortcut(showToday, wui.KeyF12)
	menu.Add(todayMenu)

	daysMenu := wui.NewMenuString(translations[settings.Language].menu.days)
	daysMenu.SetOnClick(func() {
		showView(dayView, 0)
	})
	menu.Add(daysMenu)

	weeksMenu := wui.NewMenuString(translations[settings.Language].menu.weeks)
	weeksMenu.SetOnClick(func() {
		showView(weekView, 0)
	})
	menu.Add(weeksMenu)

	monthsMenu := wui.NewMenuString(translations[settings.Language].menu.months)
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

	langMenu := wui.NewMenu(translations[settings.Language].menu.language)
	var langMenuItems [langCount]*wui.MenuString
	setLanguage := func(lang int) {
		settings.Language = lang
		for i, m := range langMenuItems {
			m.SetChecked(i == lang)
		}
		t := &translations[settings.Language]
		window.SetTitle(t.windowTitle)
		todayMenu.SetText(t.menu.today)
		daysMenu.SetText(t.menu.days)
		weeksMenu.SetText(t.menu.weeks)
		monthsMenu.SetText(t.menu.months)
		//langMenu.SetText(t.menu.language) // TODO not possible in wui right now
		showView(settings.View, 0) // updates all calendar texts
	}
	for lang := 0; lang < langCount; lang++ {
		langMenuItems[lang] = wui.NewMenuString(translations[lang].name)
		lang := lang
		langMenuItems[lang].SetOnClick(func() {
			setLanguage(lang)
		})
		langMenu.Add(langMenuItems[lang])
	}
	menu.Add(langMenu)
	setLanguage(settings.Language)

	window.Show()
}

type appSettings struct {
	Maximized bool
	View      int
	Language  int
	MonitorX  int
	MonitorY  int
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
	e.SetWordWrap(true)
	p.Add(e)
	l := wui.NewLabel()
	l.SetAlignment(wui.AlignCenter)
	p.Add(l)
	return &editor{
		panel:   p,
		caption: l,
		edit:    e,
		font:    font,
		bold:    bold,
	}
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

func formatDate(t time.Time, lang int) string {
	y, m, d := t.Date()
	weekday := t.Weekday()
	var week string
	if weekday == time.Monday {
		_, w := t.ISOWeek()
		week = fmt.Sprintf(translations[lang].calendarWeek, w)
	}
	return fmt.Sprintf(
		translations[lang].dateFormat,
		week, translations[lang].shortDays[weekday], d, m, y,
	)
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
