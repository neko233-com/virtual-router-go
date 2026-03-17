package virtual_router_server_test

import (
	"testing"
	"time"

	server "github.com/neko233-com/virtual-router-go/internal/VirtualRouterServer"
	"github.com/neko233-com/virtual-router-go/internal/config"
)

func TestServerRouterRPCStats_RankingAndKeyword(t *testing.T) {
	s := server.NewServer(&config.RouterServerConfig{RouterServerPort: 1, HTTPMonitorPort: 2})

	for i := 0; i < 5; i++ {
		s.RecordRouterRPCForTest("alpha", "beta")
	}
	for i := 0; i < 2; i++ {
		s.RecordRouterRPCForTest("gamma", "beta")
	}

	list := s.RouterRPCStats("", 10)
	if len(list) < 3 {
		t.Fatalf("expected at least 3 routers, got %d", len(list))
	}

	if list[0].RouterID != "beta" {
		t.Fatalf("expected first rank beta, got %s", list[0].RouterID)
	}
	if list[0].PerMinute < 7 {
		t.Fatalf("expected beta perMinute >= 7, got %d", list[0].PerMinute)
	}

	filtered := s.RouterRPCStats("alp", 10)
	if len(filtered) != 1 || filtered[0].RouterID != "alpha" {
		t.Fatalf("unexpected filtered result: %#v", filtered)
	}

	// push old timestamps and verify pruning works
	s.SetRouterLastMinuteHitsForTest("alpha", []int64{time.Now().Add(-2 * time.Minute).UnixMilli()})
	filtered = s.RouterRPCStats("alpha", 10)
	if filtered[0].PerMinute != 0 {
		t.Fatalf("expected pruned perMinute=0, got %d", filtered[0].PerMinute)
	}
}
