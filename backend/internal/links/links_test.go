package links

import (
	"sharelink/internal/db"
	"testing"
	"time"
)

func TestNormalizeLinkTimesConvertsToUTC(t *testing.T) {
	loc := time.FixedZone("CST", 8*60*60)
	start := time.Date(2026, 6, 21, 8, 30, 0, 0, loc)
	expire := time.Date(2026, 6, 22, 9, 0, 0, 0, loc)
	link := db.Link{
		StartTime:  &start,
		ExpireTime: &expire,
	}

	normalizeLinkTimes(&link)

	if link.StartTime == nil || link.StartTime.Location() != time.UTC {
		t.Fatalf("expected start_time to be UTC, got %#v", link.StartTime)
	}
	if got := link.StartTime.Format(time.RFC3339); got != "2026-06-21T00:30:00Z" {
		t.Fatalf("unexpected UTC start_time: %s", got)
	}
	if link.ExpireTime == nil || link.ExpireTime.Location() != time.UTC {
		t.Fatalf("expected expire_time to be UTC, got %#v", link.ExpireTime)
	}
	if got := link.ExpireTime.Format(time.RFC3339); got != "2026-06-22T01:00:00Z" {
		t.Fatalf("unexpected UTC expire_time: %s", got)
	}
}
