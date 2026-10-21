package monitor

import (
	"context"
	"io"
	"net/http"
	"net/netip"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestCheckHTTPResults(t *testing.T) {
	for _, code := range []int{200, 201, 204, 301, 404, 500} {
		client := NewCheckClient()
		client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != "GET" {
				t.Fatal(r.Method)
			}
			return &http.Response{StatusCode: code, Header: http.Header{"Location": []string{"https://example.com/redirect"}}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
		})
		got := Check(t.Context(), client, "https://example.com")
		want := "error"
		if code == 200 {
			want = "success"
		}
		if got.Status != want || got.HTTPStatus != code || got.CheckedAt.IsZero() {
			t.Fatalf("code %d: %+v", code, got)
		}
	}
}
func TestCheckErrors(t *testing.T) {
	for _, tt := range []struct {
		err  error
		want string
	}{{ErrBlockedAddress, "blocked_address"}, {context.DeadlineExceeded, "timeout"}, {io.EOF, "network_error"}} {
		client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, tt.err })}
		got := Check(t.Context(), client, "https://example.com")
		if got.Status != "error" || got.CheckError != tt.want || got.HTTPStatus != 0 {
			t.Fatalf("%+v", got)
		}
	}
}
func TestPublicIP(t *testing.T) {
	for _, ip := range []string{"127.0.0.1", "10.1.2.3", "169.254.169.254", "100.100.100.200", "192.168.1.1", "::1", "::ffff:127.0.0.1", "fc00::1", "fe80::1", "64:ff9b::7f00:1"} {
		if publicIP(netip.MustParseAddr(ip)) {
			t.Fatalf("allowed %s", ip)
		}
	}
	for _, ip := range []string{"8.8.8.8", "2606:4700:4700::1111"} {
		if !publicIP(netip.MustParseAddr(ip)) {
			t.Fatalf("blocked %s", ip)
		}
	}
	if _, err := publicDial(t.Context(), "tcp", "127.0.0.1:80"); err != ErrBlockedAddress {
		t.Fatal(err)
	}
}
