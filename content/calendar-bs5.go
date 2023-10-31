package content

import (
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/wansing/seal/content/calendar"
	"github.com/wansing/shiftpad/ical"
)

type CalendarBS5 struct {
	*ical.FeedCache
}

func (cal CalendarBS5) Handle(dirpath string, input []byte, tmpl *template.Template) error {
	_, err := tmpl.Funcs(template.FuncMap{
		"GetData": func(r *http.Request) (*calendar.Month, error) {
			year, _ := strconv.Atoi(r.URL.Query().Get("year"))
			month, _ := strconv.Atoi(r.URL.Query().Get("month"))
			return calendar.MakeMonth(cal.FeedCache, year, month)
		},
		"Link": func(r *http.Request, month calendar.Month) string {
			link := r.URL.Query()
			link.Set("year", strconv.Itoa(month.Year))
			link.Set("month", strconv.Itoa(int(month.Month)))
			r.URL.RawQuery = link.Encode()
			return r.URL.String()
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
		"NumInWeek": numInWeek,
		"NumInWeekEnd": func(end time.Time) int {
			// end time is exclusive, subtract a second if it's midnight
			if end.Hour() == 0 && end.Minute() == 0 && end.Second() == 0 && end.Nanosecond() == 0 {
				end = end.Add(-1 * time.Second)
			}
			return numInWeek(end)
		},
	}).Parse(`
		{{with GetData .Request}}
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
							<div class="p-2 text-center border-top border-dark" style="grid-column-start: calc({{NumInWeek .Begin}} + 1);">{{.Number}}</div>
						{{end}}
						{{range .Events}}
							<div class="p-2 bg-success bg-opacity-25" style="grid-column-start: calc({{NumInWeek ($week.Begin .)}} + 1); grid-column-end: calc({{NumInWeekEnd ($week.End .)}} + 2);">
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

// returns 0..6, but starting with monday
func numInWeek(t time.Time) int {
	num := int(t.Weekday()) - 1
	if num < 0 {
		num += 7
	}
	return num
}
