package auth

import (
	"strings"
	"testing"
	"time"
)

func TestProfileValidation(t *testing.T) {
	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	birthday := "2000-02-29"
	name, date, err := validateProfile("  Иванов Иван Иванович  ", &birthday, now)
	if err != nil || name != "Иванов Иван Иванович" || date.Format("2006-01-02") != birthday {
		t.Fatalf("%q %v %v", name, date, err)
	}
	for _, input := range []string{"2001-02-29", "2026-10-15", "0000-01-01", "14.10.2000", "2000-01-01T00:00:00Z"} {
		if _, _, err := validateProfile("Иван", &input, now); err == nil {
			t.Fatalf("accepted birthday %q", input)
		}
	}
	for _, input := range []string{"", "   ", "\t\n", "\u00a0\u2003", "12345", "---...", "\u200b", strings.Repeat("я", 201), "Иван\nИванов", string([]byte{0xff})} {
		if _, _, err := validateProfile(input, nil, now); err == nil {
			t.Fatalf("accepted name %q", input)
		}
	}
	if _, date, err := validateProfile(strings.Repeat("я", 200), nil, now); err != nil || date != nil {
		t.Fatal("optional birthday or Unicode name rejected")
	}
	empty := ""
	if name, date, err := validateProfile("Анна-Мария О’Коннор", &empty, now); err != nil || name != "Анна-Мария О’Коннор" || date != nil {
		t.Fatal("valid name or clearing birthday rejected")
	}
}
