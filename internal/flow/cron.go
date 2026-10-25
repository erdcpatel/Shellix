package flow

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// CronSchedule represents parsed cron matching rules
type CronSchedule struct {
	Minutes map[int]bool
	Hours   map[int]bool
	Days    map[int]bool
	Months  map[int]bool
	Weekdays map[int]bool
	RawExpr string
}

// ParseCronExpression parses standard 5-part cron expressions into a schedule
func ParseCronExpression(expr string) (*CronSchedule, error) {
	clean := strings.TrimSpace(expr)

	// Expand standard macro aliases
	switch clean {
	case "@hourly":
		clean = "0 * * * *"
	case "@daily", "@midnight":
		clean = "0 0 * * *"
	case "@weekly":
		clean = "0 0 * * 0"
	case "@monthly":
		clean = "0 0 1 * *"
	case "@yearly", "@annually":
		clean = "0 0 1 1 *"
	}

	fields := strings.Fields(clean)
	if len(fields) != 5 {
		return nil, fmt.Errorf("invalid cron expression: expected 5 fields, got %d in %q", len(fields), expr)
	}

	mins, err := parseCronField(fields[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("invalid minute field: %w", err)
	}

	hours, err := parseCronField(fields[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("invalid hour field: %w", err)
	}

	days, err := parseCronField(fields[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("invalid day-of-month field: %w", err)
	}

	months, err := parseCronField(fields[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("invalid month field: %w", err)
	}

	weekdays, err := parseCronField(fields[4], 0, 6)
	if err != nil {
		return nil, fmt.Errorf("invalid day-of-week field: %w", err)
	}

	return &CronSchedule{
		Minutes:  mins,
		Hours:    hours,
		Days:     days,
		Months:   months,
		Weekdays: weekdays,
		RawExpr:  expr,
	}, nil
}

func parseCronField(field string, min, max int) (map[int]bool, error) {
	res := make(map[int]bool)

	// Comma-separated parts
	parts := strings.Split(field, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Step format: */step or min-max/step
		if strings.Contains(part, "/") {
			subParts := strings.SplitN(part, "/", 2)
			step, err := strconv.Atoi(subParts[1])
			if err != nil || step <= 0 {
				return nil, fmt.Errorf("invalid step value in %s", part)
			}

			start := min
			end := max
			if subParts[0] != "*" {
				if strings.Contains(subParts[0], "-") {
					rangeParts := strings.SplitN(subParts[0], "-", 2)
					start, _ = strconv.Atoi(rangeParts[0])
					end, _ = strconv.Atoi(rangeParts[1])
				} else {
					start, _ = strconv.Atoi(subParts[0])
				}
			}

			for i := start; i <= end; i += step {
				if i >= min && i <= max {
					res[i] = true
				}
			}
			continue
		}

		// Wildcard: *
		if part == "*" {
			for i := min; i <= max; i++ {
				res[i] = true
			}
			continue
		}

		// Range format: a-b
		if strings.Contains(part, "-") {
			rangeParts := strings.SplitN(part, "-", 2)
			start, err1 := strconv.Atoi(rangeParts[0])
			end, err2 := strconv.Atoi(rangeParts[1])
			if err1 != nil || err2 != nil || start > end {
				return nil, fmt.Errorf("invalid range %s", part)
			}
			for i := start; i <= end; i++ {
				if i >= min && i <= max {
					res[i] = true
				}
			}
			continue
		}

		// Single integer value
		val, err := strconv.Atoi(part)
		if err != nil || val < min || val > max {
			return nil, fmt.Errorf("value %s out of range [%d-%d]", part, min, max)
		}
		res[val] = true
	}

	if len(res) == 0 {
		return nil, fmt.Errorf("empty match set for field %s", field)
	}

	return res, nil
}

// NextTrigger calculates the next timestamp matching the schedule after the given time
func (cs *CronSchedule) NextTrigger(after time.Time) time.Time {
	// Truncate to second 0 and advance by 1 minute
	cur := after.Truncate(time.Minute).Add(time.Minute)

	// Scan up to 5 years (5 * 365 * 24 * 60 minutes)
	limit := cur.Add(5 * 365 * 24 * time.Hour)

	for cur.Before(limit) {
		if !cs.Months[int(cur.Month())] {
			// Advance to next month
			cur = time.Date(cur.Year(), cur.Month()+1, 1, 0, 0, 0, 0, cur.Location())
			continue
		}

		if !cs.Days[cur.Day()] || !cs.Weekdays[int(cur.Weekday())] {
			// Advance to next day
			cur = time.Date(cur.Year(), cur.Month(), cur.Day()+1, 0, 0, 0, 0, cur.Location())
			continue
		}

		if !cs.Hours[cur.Hour()] {
			// Advance to next hour
			cur = time.Date(cur.Year(), cur.Month(), cur.Day(), cur.Hour()+1, 0, 0, 0, cur.Location())
			continue
		}

		if cs.Minutes[cur.Minute()] {
			return cur
		}

		cur = cur.Add(time.Minute)
	}

	return time.Time{}
}
