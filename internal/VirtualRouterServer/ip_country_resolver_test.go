package VirtualRouterServer

import "testing"

func TestResolveIPCountry_LocalAndPrivate(t *testing.T) {
	if got := resolveIPCountry("127.0.0.1"); got != "本机 (LOCAL)" {
		t.Fatalf("expected 本机, got %q", got)
	}
	if got := resolveIPCountry("10.0.0.2"); got != "内网 (LAN)" {
		t.Fatalf("expected 内网, got %q", got)
	}
	if got := resolveIPCountry("not-an-ip"); got != "未知" {
		t.Fatalf("expected 未知 for invalid ip, got %q", got)
	}
}
