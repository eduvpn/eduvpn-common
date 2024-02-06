package server

import "time"

// RenewButtonTime returns the time when the renew button should be shown for the server
// Implemented according to: https://github.com/eduvpn/documentation/blob/cdf4d054f7652d74e4192494e8bb0e21040e46ac/API.md#session-expiry
func RenewButtonTime(st time.Time, et time.Time) int64 {
	d := et.Sub(st)

	// If the time is less than 24 hours (a day left), show it when 30 minutes have passed or on expired if less than 30 minutes
	dayl := time.Duration(24 * time.Hour)
	if d < dayl {
		// Get the minimum time to add, 30 minutes or on expired
		m := time.Duration(30 * time.Minute)
		// The total delta time is larger, return that we should show the button after 30 minutes
		if d > m {
			return st.Add(30 * time.Minute).Unix()
		}
		// Just show it on expired
		return st.Add(d).Unix()
	}

	// Else just show it when 24 hours is left
	// This is the delta minus 24 hours left as that's how long it takes for a day to be left in the expiry
	// We thus add this to the start time
	tillDay := d - dayl
	t := st.Add(tillDay)
	return t.Unix()
}

func CountdownTime(st time.Time, et time.Time) int64 {
	d := et.Sub(st)

	dayl := time.Duration(24 * time.Hour)

	// This is just the last 24 hours
	// if less than or equal to 24 hours, immediately
	if d <= dayl {
		return st.Unix()
	}

	tillDay := d - dayl
	t := st.Add(tillDay)
	return t.Unix()
}

func NotificationTimes(st time.Time, et time.Time) []int64 {
	last := []time.Duration{
		time.Duration(0),
		time.Duration(1 * time.Hour),
		time.Duration(2 * time.Hour),
		time.Duration(4 * time.Hour),
	}

	var t []int64

	d := et.Sub(st)
	for _, l := range last {
		// If the notification remaining time is more than the total delta, continue
		if l > d {
			continue
		}
		// calculating the time till a notification must happen
		tillN := d - l
		// Get absolute time when this notification must be shown by adding the delta
		c := st.Add(tillN)
		t = append(t, c.Unix())
	}
	return t
}
