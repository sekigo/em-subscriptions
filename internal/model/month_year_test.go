package model

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseMonthYear(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    time.Time
		wantErr bool
	}{
		{"valid july 2025", "07-2025", time.Date(2025, time.July, 1, 0, 0, 0, 0, time.UTC), false},
		{"valid january 2000", "01-2000", time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC), false},
		{"trims whitespace", "  12-2030  ", time.Date(2030, time.December, 1, 0, 0, 0, 0, time.UTC), false},
		{"invalid month", "13-2025", time.Time{}, true},
		{"swapped order", "2025-07", time.Time{}, true},
		{"slash separator", "07/2025", time.Time{}, true},
		{"empty", "", time.Time{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMonthYear(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !got.Time().Equal(tt.want) {
				t.Fatalf("got %v, want %v", got.Time(), tt.want)
			}
		})
	}
}

func TestMonthYearJSONRoundTrip(t *testing.T) {
	original := NewMonthYear(2025, time.July)
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `"07-2025"` {
		t.Fatalf("marshal = %s, want \"07-2025\"", data)
	}

	var back MonthYear
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if !back.Time().Equal(original.Time()) {
		t.Fatalf("round-trip mismatch: %v vs %v", back, original)
	}
}

func TestMonthYearJSONNull(t *testing.T) {
	var m MonthYear
	if err := json.Unmarshal([]byte(`null`), &m); err != nil {
		t.Fatal(err)
	}
	if !m.IsZero() {
		t.Fatalf("null should produce zero value, got %v", m)
	}
}
