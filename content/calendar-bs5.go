package content

import (
	"html/template"
	"net/url"
	"strconv"
	"time"

	"github.com/wansing/go-ical-cache"
	"github.com/wansing/seal"
	"github.com/wansing/seal/content/calendar"
)

type CalendarBS5 struct {
	Config icalcache.Config // default url: file content
}

type monthView struct {
	calendar.Month
	Error error
}

type calendarData struct {
	Feed     *icalcache.Cache
	Fileroot string
}

func (data calendarData) Link(requestURL *url.URL, month calendar.Month) string {
	var u = *requestURL // copy
	link := u.Query()
	link.Set("year", strconv.Itoa(month.Year))
	link.Set("month", strconv.Itoa(int(month.Month)))
	u.RawQuery = link.Encode()
	u.Fragment = data.Fileroot // anchor
	return u.String()
}

func (data calendarData) Month(requestURL *url.URL) monthView {
	year, _ := strconv.Atoi(requestURL.Query().Get("year"))
	month, _ := strconv.Atoi(requestURL.Query().Get("month"))
	events, err := data.Feed.Get(time.Local)
	return monthView{
		Month: calendar.MakeMonth(events, year, month),
		Error: err,
	} // don't return err, don't interrupt template execution
}

func (calendarData) MonthName(month time.Month) string {
	switch month {
	case time.January:
		return "Januar"
	case time.February:
		return "Februar"
	case time.March:
		return "März"
	case time.April:
		return "April"
	case time.May:
		return "Mai"
	case time.June:
		return "Juni"
	case time.July:
		return "Juli"
	case time.August:
		return "August"
	case time.September:
		return "September"
	case time.October:
		return "Oktober"
	case time.November:
		return "November"
	case time.December:
		return "Dezember"
	default:
		return ""
	}
}

func (cal CalendarBS5) Make(t *template.Template, urlpath, fileroot string, filecontent []byte) error {
	var config = cal.Config
	if config == (icalcache.Config{}) {
		config = icalcache.Config{
			URL: string(filecontent),
		}
	}

	dataFuncName := seal.TemplateName(urlpath, fileroot)
	_, err := t.Funcs(template.FuncMap{
		dataFuncName: func() calendarData {
			return calendarData{
				Feed:     &icalcache.Cache{Config: config},
				Fileroot: fileroot,
			}
		},
	}).Parse(`
		{{$data := ` + dataFuncName + `}}
		{{with $data.Month .RequestURL}}
			<div>
				{{with .Error}}
					<div class="alert alert-danger text-center">Error getting calendar events: {{.}}</div>
				{{end}}
				<div class="p-2 d-flex justify-content-center align-items-center">
					<a class="btn btn-success" href="{{$data.Link $.RequestURL .Prev}}">&#9668;</a>
					<strong class="h3 mx-3 my-0">{{$data.MonthName .Month.Month}} {{.Year}}</strong>
					<a class="btn btn-success" href="{{$data.Link $.RequestURL .Next}}">&#9658;</a>
				</div>
				<div style="display: grid; grid-template-columns: repeat(7, 1fr);">
					<div class="p-2 text-center"><strong>Mo</strong></div>
					<div class="p-2 text-center"><strong>Di</strong></div>
					<div class="p-2 text-center"><strong>Mi</strong></div>
					<div class="p-2 text-center"><strong>Do</strong></div>
					<div class="p-2 text-center"><strong>Fr</strong></div>
					<div class="p-2 text-center"><strong>Sa</strong></div>
					<div class="p-2 text-center"><strong>So</strong></div>
				</div>
				<div class="border-bottom border-dark" style="display: grid; grid-template-columns: repeat(7, 1fr);">
					{{range .Weeks}}
						{{$week := .}}
						{{range .Days}}
							<div class="p-2 text-center border-top border-dark" style="grid-column-start: calc({{.NumInWeek}} + 1);">{{.Number}}</div>
						{{end}}
						{{range .Events}}
							<div class="p-2 bg-success bg-opacity-25" style="grid-column-start: calc({{$week.NumInWeekBegin .}} + 1); grid-column-end: calc({{$week.NumInWeekEnd .}} + 2);">
								{{with .URL}}
									<a href="{{.}}">
								{{end}}
								{{.Summary}}
								{{if .URL}}
									</a>
								{{end}}
							</div>
						{{end}}
					{{end}}
				</div>
			</div>
		{{end}}
	`)
	return err
}
