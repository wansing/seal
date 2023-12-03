package content

import (
	"errors"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/icza/gox/timex"
)

type countdownData struct {
	End     time.Time
	Years   int
	Months  int
	Days    int
	Hours   int
	Minutes int
	Seconds int
}

func Countdown(dirpath string, input []byte, tmpl *template.Template) error {
	isoEnd, tmplHtml, _ := strings.Cut(string(input), "\n")

	isoEnd = strings.TrimSpace(isoEnd)
	if isoEnd == "" {
		return errors.New("missing end time")
	}
	end, err := time.Parse("2006-01-02 15:04:05 -0700", isoEnd)
	if err != nil {
		return fmt.Errorf("parsing time: %v", err)
	}

	tmplHtml = strings.TrimSpace(tmplHtml)
	if tmplHtml == "" {
		tmplHtml = `
			<span id="years">  {{$years  }}</span> years,
			<span id="months"> {{$months }}</span> months,
			<span id="days">   {{$days   }}</span> days,
			<span id="hours">  {{$hours  }}</span> hours,
			<span id="minutes">{{$minutes}}</span> minutes,
			<span id="seconds">{{$seconds}}</span> seconds`
	}

	_, err = tmpl.Funcs(template.FuncMap{
		"Countdown": func() countdownData {
			years, months, days, hours, minutes, seconds := timex.Diff(time.Now(), end) // respects leap years
			return countdownData{
				End:     end,
				Years:   years,
				Months:  months,
				Days:    days,
				Hours:   hours,
				Minutes: minutes,
				Seconds: seconds,
			}
		},
	}).Parse(`
		{{$data := Countdown}}

		<script type="text/javascript">
			function updateCountdown() {

				// Copyright 2018 Andras Belicza, Apache License 2.0
				// https://github.com/icza/gox/blob/master/timex/timex.go
				// modifications: translated from Golang to JavaScript

				let a = new Date();
				let b = new Date({{$data.End.Unix}} * 1000); // constructor takes milliseconds
				if(a > b) {
					return;
				}

				let years  = b.getFullYear() - a.getFullYear();
				let months = b.getMonth()    - a.getMonth();
				let days   = b.getDate()     - a.getDate();
				let hours  = b.getHours()    - a.getHours();
				let mins   = b.getMinutes()  - a.getMinutes();
				let secs   = b.getSeconds()  - a.getSeconds();

				if(secs < 0) {
					secs += 60;
					mins--;
				}

				if(mins < 0) {
					mins += 60;
					hours--;
				}

				if(hours < 0) {
					hours += 24;
					days--;
				}

				if(days < 0) {
					let t = new Date(a.getFullYear(), a.getMonth(), 32, 0, 0, 0);
					days += 32 - t.getDate();
					months--;
				}

				if(months < 0) {
					months += 12;
					years--;
				}

				// end of afore-mentioned copyright

				elementYears = document.getElementById("years");
				if(elementYears) {
					elementYears.innerHTML = years;
				}

				elementMonths = document.getElementById("months");
				if(elementMonths) {
					elementMonths.innerHTML = months;
				}

				elementDays = document.getElementById("days");
				if(elementDays) {
					elementDays.innerHTML = days;
				}

				elementHours = document.getElementById("hours");
				if(elementHours) {
					elementHours.innerHTML = hours;
				}

				elementMinutes = document.getElementById("minutes");
				if(elementMinutes) {
					elementMinutes.innerHTML = mins;
				}

				elementSeconds = document.getElementById("seconds");
				if(elementSeconds) {
					elementSeconds.innerHTML = secs;
				}

				setTimeout(updateCountdown, 1000);
			}
			updateCountdown();
		</script>

		{{$years   := $data.Years}}
		{{$months  := $data.Months}}
		{{$days    := $data.Days}}
		{{$hours   := $data.Hours}}
		{{$minutes := $data.Minutes}}
		{{$seconds := $data.Seconds}}

		` + tmplHtml)

	return err
}
