package services

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strings"
	"testing"

	"github.com/bnhminh1010/homelab-dashboard/internal/model"
)

func TestValidateDNSURLAcceptsValidFormsAndRejectsUnsafe(t *testing.T) {
	for _, value := range []string{
		"dns://100.64.0.10/google.com",
		"dns://[fd7a:115c:a1e0::1]:5353/example.com",
		"dns://pihole.local/ads.example.com?type=A",
		"dns://10.0.0.2:53/test.local?type=AAAA",
	} {
		parsed, err := url.Parse(value)
		if err != nil {
			t.Fatalf("failed to parse %q: %v", value, err)
		}
		if err := validateDNSURL(parsed); err != nil {
			t.Fatalf("valid DNS probe %q rejected: %v", value, err)
		}
	}

	for _, value := range []string{
		"dns://",
		"dns://10.0.0.2",
		"dns://user:pass@10.0.0.2/domain.com",
		"dns://10.0.0.2/domain.com#frag",
		"dns://10.0.0.2:0/domain.com",
		"dns://10.0.0.2:70000/domain.com",
		"dns://10.0.0.2/domain.com?type=MX",
		"dns://10.0.0.2/" + strings.Repeat("a", 254),
	} {
		parsed, _ := url.Parse(value)
		if err := validateDNSURL(parsed); err == nil {
			t.Fatalf("unsafe or invalid DNS probe %q accepted", value)
		}
	}
}

func TestProberDNSProbeUsesPolicyDialer(t *testing.T) {
	var dialedNetwork, dialedAddress string
	prober := &Prober{dialContext: func(_ context.Context, network, address string) (net.Conn, error) {
		dialedNetwork, dialedAddress = network, address
		return nil, errors.New("mock dial error")
	}}
	result := prober.Probe(context.Background(), "dns://10.0.0.2/google.com")
	if result.Status != model.ServiceStatusDown {
		t.Fatalf("DNS probe status = %s, want down", result.Status)
	}
	if dialedAddress != "10.0.0.2:53" {
		t.Fatalf("DNS dial = %s %s, want network(udp/tcp) 10.0.0.2:53", dialedNetwork, dialedAddress)
	}
}
