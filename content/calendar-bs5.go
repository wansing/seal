package content

import (
	"html/template"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/wansing/seal"
	"github.com/wansing/seal/content/calendar"
	"github.com/wansing/shiftpad/ical"
)

type CalendarBS5 struct {
	Anchor string      // for prev, next button link, default: filename
	Config ical.Config // default url: file content
}

func (cal CalendarBS5) Parse(dir *seal.Dir, filestem string, filecontent []byte) error {
	var anchor = cal.Anchor
	if anchor == "" {
		anchor = filepath.Base(dir.URLPath)
	}

	var config = cal.Config
	if config == (ical.Config{}) {
		config = ical.Config{
			URL: string(filecontent),
		}
	}
	feed := &ical.FeedCache{
		Config: config,
	}

	_, err := dir.Template.New(filestem).Funcs(template.FuncMap{
		"CalendarBS5": func(r *http.Request) (*calendar.Month, error) {
			year, _ := strconv.Atoi(r.URL.Query().Get("year"))
			month, _ := strconv.Atoi(r.URL.Query().Get("month"))
			events, err := feed.Get(time.Local)
			if err != nil {
				return nil, err
			}
			return calendar.MakeMonth(events, year, month)
		},
		"Link": func(r *http.Request, month calendar.Month) string {
			var url = *r.URL // copy
			link := url.Query()
			link.Set("year", strconv.Itoa(month.Year))
			link.Set("month", strconv.Itoa(int(month.Month)))
			url.RawQuery = link.Encode()
			url.Fragment = anchor
			return url.String()
		},
		"MonthName": func(month time.Month) string {
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
		},
	}).Parse(`
		{{with CalendarBS5 .Request}}
			<div>
				<div class="p-2 d-flex justify-content-center align-items-center">
					<a class="btn btn-success" href="{{Link $.Request .Prev}}">&#9668;</a>
					<strong class="h3 mx-3 my-0">{{MonthName .Month}} {{.Year}}</strong>
					<a class="btn btn-success" href="{{Link $.Request .Next}}">&#9658;</a>
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
