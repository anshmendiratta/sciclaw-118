package main

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestShouldOfferConfigHealthRepair(t *testing.T) {
	cases := map[string]bool{
		"app":     true,
		"tui":     true,
		"gateway": true,
		"service": true,
		"status":  true,
		"agent":   true,
		"routing": false,
		"doctor":  false,
		"auth":    false,
	}
	for cmd, want := range cases {
		if got := shouldOfferConfigHealthRepair(cmd); got != want {
			t.Fatalf("shouldOfferConfigHealthRepair(%q)=%v want %v", cmd, got, want)
		}
	}
}

func TestRoutingMentionMismatchIndexes(t *testing.T) {
	f := false
	cfg := config.DefaultConfig()
	cfg.Routing.Mappings = []config.RoutingMapping{
		{Channel: "discord", ChatID: "1", RequireMention: &f},
		{Channel: "discord", ChatID: "2"},
		{Channel: "telegram", ChatID: "3", RequireMention: &f},
	}
	got := routingMentionMismatchIndexes(cfg)
	if len(got) != 1 || got[0] != 0 {
		t.Fatalf("unexpected mismatch indexes: %#v", got)
	}
}

func TestCollectDiscordRoutingSendersSortedUnique(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Routing.Mappings = []config.RoutingMapping{
		{Channel: "discord", AllowedSenders: config.FlexibleStringSlice{"2", "1", "2"}},
		{Channel: "telegram", AllowedSenders: config.FlexibleStringSlice{"3"}},
		{Channel: "discord", AllowedSenders: config.FlexibleStringSlice{"3", "1"}},
	}
	got := collectDiscordRoutingSenders(cfg)
	want := []string{"1", "2", "3"}
	if len(got) != len(want) {
		t.Fatalf("senders len=%d want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("senders[%d]=%q want %q (%#v)", i, got[i], want[i], got)
		}
	}
}

func TestDetectConfigHealthIssues(t *testing.T) {
	f := false
	cfg := config.DefaultConfig()
	cfg.Routing.Enabled = true
	cfg.Routing.UnmappedBehavior = config.RoutingUnmappedBehaviorDefault
	cfg.Routing.Mappings = []config.RoutingMapping{
		{
			Channel:        "discord",
			ChatID:         "chan-1",
			AllowedSenders: config.FlexibleStringSlice{"u2", "u1"},
			RequireMention: &f,
		},
	}
	cfg.Channels.Discord.Enabled = true
	cfg.Channels.Discord.AllowFrom = config.FlexibleStringSlice{}

	issues := detectConfigHealthIssues(cfg)
	if !issues.hasAny() {
		t.Fatal("expected issues to be detected")
	}
	if len(issues.discordMentionMismatch) != 1 || issues.discordMentionMismatch[0] != 0 {
		t.Fatalf("unexpected mention mismatch indexes: %#v", issues.discordMentionMismatch)
	}
	if !issues.unmappedBehaviorLegacy {
		t.Fatal("expected unmapped behavior mismatch")
	}
	if !issues.discordAllowlistEmpty {
		t.Fatal("expected empty discord allowlist mismatch")
	}
	if len(issues.suggestedDiscordUsers) != 2 || issues.suggestedDiscordUsers[0] != "u1" || issues.suggestedDiscordUsers[1] != "u2" {
		t.Fatalf("unexpected suggested users: %#v", issues.suggestedDiscordUsers)
	}
}

func TestApplyRoutingMentionRequired(t *testing.T) {
	f := false
	cfg := config.DefaultConfig()
	cfg.Routing.Mappings = []config.RoutingMapping{
		{Channel: "discord", ChatID: "1", RequireMention: &f},
		{Channel: "discord", ChatID: "2"},
	}
	n := applyRoutingMentionRequired(cfg, []int{0, 1, 99, -1})
	if n != 2 {
		t.Fatalf("updated=%d want 2", n)
	}
	for i, m := range cfg.Routing.Mappings {
		if m.RequireMention == nil || !*m.RequireMention {
			t.Fatalf("mapping %d require_mention not true", i)
		}
	}
}
