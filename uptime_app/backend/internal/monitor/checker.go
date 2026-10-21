package monitor

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"time"
)

const CheckTimeout = 10 * time.Second

var ErrBlockedAddress = errors.New("monitor target must use a public IP address")

// Dial the validated IP itself, so DNS cannot change between validation and connection.
func publicDial(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, ErrBlockedAddress
	}
	for _, ip := range ips {
		if !publicIP(ip) {
			return nil, ErrBlockedAddress
		}
	}
	var last error
	for _, ip := range ips {
		conn, err := (&net.Dialer{Timeout: CheckTimeout}).DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
		last = err
	}
	return nil, last
}

func publicIP(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return false
	}
	for _, prefix := range []string{"100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4", "2001:db8::/32", "2001::/32", "2002::/16", "64:ff9b::/96", "64:ff9b:1::/48"} {
		if netip.MustParsePrefix(prefix).Contains(ip) {
			return false
		}
	}
	return true
}

func NewCheckClient() *http.Client {
	return &http.Client{
		Timeout: CheckTimeout,
		// A redirect is itself a non-200 result; never follow it into another network.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		Transport:     &http.Transport{DialContext: publicDial, DisableKeepAlives: true, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: CheckTimeout, MaxResponseHeaderBytes: 64 << 10},
	}
}

type CheckResult struct {
	CheckedAt  time.Time
	HTTPStatus int
	CheckError string
	Status     string
}

func Check(ctx context.Context, client *http.Client, target string) CheckResult {
	ctx, cancel := context.WithTimeout(ctx, CheckTimeout)
	defer cancel()
	result := CheckResult{Status: "error"}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		result.CheckError = "invalid_url"
		result.CheckedAt = time.Now().UTC()
		return result
	}
	req.Header.Set("User-Agent", "UptimeToolkit/1.0")
	resp, err := client.Do(req)
	result.CheckedAt = time.Now().UTC()
	if err != nil {
		var networkError net.Error
		switch {
		case errors.Is(err, ErrBlockedAddress):
			result.CheckError = "blocked_address"
		case errors.Is(err, context.DeadlineExceeded) || errors.As(err, &networkError) && networkError.Timeout():
			result.CheckError = "timeout"
		default:
			result.CheckError = "network_error"
		}
		return result
	}
	defer resp.Body.Close()
	result.HTTPStatus = resp.StatusCode
	if resp.StatusCode == http.StatusOK {
		result.Status = "success"
	}
	return result
}
