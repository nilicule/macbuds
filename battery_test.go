package main

import "testing"

func TestLowBatteryAlert(t *testing.T) {
	tests := []struct {
		name         string
		percent      int
		announced    int
		wantAlert    int
		wantAnnounce int
	}{
		{"healthy, nothing announced", 80, 0, 0, 0},
		{"exactly above top threshold", 21, 0, 0, 0},
		{"first drop to 20", 20, 0, 20, 20},
		{"drop below 20", 15, 0, 20, 20},
		{"still in the 20 band, already announced", 15, 20, 0, 20},
		{"drop to the 10 band", 10, 20, 10, 10},
		{"below 10, already announced", 5, 10, 0, 10},
		{"zero percent", 0, 10, 0, 10},
		{"skips straight past 20 to 10", 8, 0, 10, 10},
		// Recharging re-arms lower thresholds without notifying.
		{"charged from 10 band into 20 band", 15, 10, 0, 20},
		{"charged clear of all thresholds", 90, 10, 0, 0},
		{"drains again after re-arming", 9, 20, 10, 10},
		{"re-alerts at 20 after a full recharge", 19, 0, 20, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alert, updated := lowBatteryAlert(tt.percent, tt.announced)
			if alert != tt.wantAlert || updated != tt.wantAnnounce {
				t.Errorf("lowBatteryAlert(%d, %d) = (%d, %d), want (%d, %d)",
					tt.percent, tt.announced, alert, updated, tt.wantAlert, tt.wantAnnounce)
			}
		})
	}
}

// A drain from full to empty should notify exactly once per threshold.
func TestLowBatteryAlertDrainCycle(t *testing.T) {
	announced := 0
	var alerts []int
	for _, p := range []int{100, 80, 45, 25, 20, 18, 12, 10, 7, 3, 0} {
		alert, updated := lowBatteryAlert(p, announced)
		announced = updated
		if alert > 0 {
			alerts = append(alerts, alert)
		}
	}
	want := []int{20, 10}
	if len(alerts) != len(want) {
		t.Fatalf("got alerts %v, want %v", alerts, want)
	}
	for i := range want {
		if alerts[i] != want[i] {
			t.Fatalf("got alerts %v, want %v", alerts, want)
		}
	}
}
