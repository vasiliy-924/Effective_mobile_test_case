package month

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	got, err := Parse("07-2025")
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("Parse(07-2025) = %v, want %v", got, want)
	}
}

func TestParse_invalid(t *testing.T) {
	_, err := Parse("13-2025")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFormat_roundTrip(t *testing.T) {
	in := "01-2024"
	d, err := Parse(in)
	if err != nil {
		t.Fatal(err)
	}
	if Format(d) != in {
		t.Fatalf("Format = %q want %q", Format(d), in)
	}
}

func TestMonthCountInclusive(t *testing.T) {
	from := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)
	if n := MonthCountInclusive(from, to); n != 3 {
		t.Fatalf("count = %d want 3", n)
	}
	if n := MonthCountInclusive(to, from); n != 0 {
		t.Fatalf("reversed count = %d want 0", n)
	}
	if n := MonthCountInclusive(from, from); n != 1 {
		t.Fatalf("same month = %d want 1", n)
	}
}
