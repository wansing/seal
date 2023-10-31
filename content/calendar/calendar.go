package calendar

import (
	"time"

	"github.com/wansing/shiftpad/ical"
)

type Month struct {
	Year  int
	Month time.Month
	Weeks []Week
}

func (month Month) Next() Month {
	t := time.Date(month.Year, month.Month, 1, 0, 0, 0, 0, time.Local).AddDate(0, 1, 0)
	return Month{
		Year:  t.Year(),
		Month: t.Month(),
	}
}

func (month Month) Prev() Month {
	t := time.Date(month.Year, month.Month, 1, 0, 0, 0, 0, time.Local).AddDate(0, -1, 0)
	return Month{
		Year:  t.Year(),
		Month: t.Month(),
	}
}

func MakeMonth(proxy *ical.FeedCache, year, month int) (*Month, error) {
	// check arguments
	if year <= 0 {
		year = time.Now().Year()
	}
	if month < 1 || month > 12 {
		month = int(time.Now().Month())
	}

	// get begin and end of month
	begin := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	end := begin.AddDate(0, 1, 0)

	// go back to monday of first week
	for begin.Weekday() != time.Monday {
		begin = begin.AddDate(0, 0, -1)
	}

	// go forth to monday of next week
	for end.Weekday() != time.Monday {
		end = end.AddDate(0, 0, 1)
	}

	// get and pre-filter events
	events, err := proxy.Get(time.Local)
	if err != nil {
		return nil, err
	}
	events = filterEvents(events, begin, end)

	// make weeks
	var weeks []Week
	for ; begin.Before(end); begin = begin.AddDate(0, 0, 7) {
		_, weekNumber := begin.ISOWeek()
		var week = Week{
			Number: weekNumber,
			Events: filterEvents(events, begin, begin.AddDate(0, 0, 7)),
		}
		for i := 0; i < 7; i++ {
			week.Days[i] = Day{
				Begin: begin.AddDate(0, 0, i),
			}
		}
		weeks = append(weeks, week)
	}

	return &Month{
		Year:  year,
		Month: time.Month(month),
		Weeks: weeks,
	}, nil
}

type Week struct {
	Number int
	Days   [7]Day
	Events []ical.Event
}

func (week Week) Begin(event ical.Event) time.Time {
	if beginOfWeek := week.Days[0].Begin; event.Start.Before(beginOfWeek) {
		return beginOfWeek
	}
	return event.Start
}

func (week Week) End(event ical.Event) time.Time {
	if endOfWeek := week.Days[6].End(); event.End.After(endOfWeek) {
		return endOfWeek
	}
	return event.End
}

type Day struct {
	Begin time.Time
}

// exclusive
func (day Day) End() time.Time {
	return day.Begin.AddDate(0, 0, 1)
}

func (day Day) Number() int {
	return day.Begin.Day()
}

func filterEvents(events []ical.Event, begin, end time.Time) []ical.Event {
	var result []ical.Event
	for _, event := range events {
		if overlaps(event.Start, event.End, begin, end) {
			result = append(result, event)
		}
	}
	return result
}

func overlaps(begin1, end1, begin2, end2 time.Time) bool {
	// end is always considered exclusive, so we check equality too
	if end1.Before(begin2) || end1.Equal(begin2) {
		return false
	}
	if begin1.After(end2) || begin1.Equal(end2) {
		return false
	}
	return true
}
