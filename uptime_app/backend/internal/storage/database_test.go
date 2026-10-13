package storage

import "testing"

func TestInvalidDriverOptions(t *testing.T) {
	for _, dsn := range []string{"postgres://localhost/test?sslmode=unknown", "postgres://localhost/test?dial_timeout=bad"} {
		if _, err := newConnector(dsn); err == nil {
			t.Fatalf("accepted invalid driver DSN")
		}
	}
}
