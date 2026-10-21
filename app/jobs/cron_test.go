package jobs

import (
	"github.com/lisijie/cron"
	"testing"
	"time"
)

func TestGetEntriesSkipsSystemJobsBeforeLimiting(t *testing.T) {
	old := mainCron
	mainCron = cron.New()
	defer func() { mainCron = old }()
	if err := mainCron.AddFunc("0 * * * * *", func() {}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int{1, 2} {
		if err := mainCron.AddJob("0 * * * * *", NewCommandJob(id, "test", "echo unused")); err != nil {
			t.Fatal(err)
		}
	}
	entries := GetEntries(1)
	if len(entries) != 1 {
		t.Fatalf("got %d entries", len(entries))
	}
	if _, ok := entries[0].Job.(*Job); !ok {
		t.Fatal("system entry leaked")
	}
	if got := len(GetEntries(30)); got != 2 {
		t.Fatalf("got %d task entries", got)
	}
}

func TestFailureMailIncludesTimeoutWithoutCommandError(t *testing.T) {
	for _, tc := range []struct {
		notify                int
		failed, timeout, want bool
	}{
		{0, true, true, false}, {1, false, false, false}, {1, true, false, true}, {1, false, true, true}, {2, false, false, true},
	} {
		if got := shouldNotify(tc.notify, tc.failed, tc.timeout); got != tc.want {
			t.Errorf("notify=%d failed=%v timeout=%v got %v", tc.notify, tc.failed, tc.timeout, got)
		}
	}
}

func TestCleanupRunsDailyAtFive(t *testing.T) {
	schedule, err := cron.Parse(cleanupCronSpec)
	if err != nil {
		t.Fatal(err)
	}
	zone := time.FixedZone("CST", 8*60*60)
	for _, start := range []time.Time{time.Date(2026, 10, 21, 4, 59, 59, 0, zone), time.Date(2026, 10, 21, 5, 0, 0, 0, zone)} {
		next := schedule.Next(start)
		if next.Hour() != 5 || next.Minute() != 0 || next.Second() != 0 {
			t.Fatalf("unexpected next run %v", next)
		}
		if start.Hour() == 5 && next.Day() != 22 {
			t.Fatalf("cleanup repeated on same day: %v", next)
		}
	}
}
