package events

import (
	"testing"

	"github.com/PastureStack/compose-cli/internal/rancherclient/v2"
)

func TestNewEventRouterBuildsBoundedSubscriptionURL(t *testing.T) {
	router, err := NewEventRouter(
		"compose", 1, "https://api.example.test/v2-beta?token=secret#fragment",
		"access", "secret", &client.RancherClient{}, nil, "resource", 1, PingConfig{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if router.subscribeURL != "wss://api.example.test/v2-beta/subscribe" {
		t.Fatalf("unexpected subscription URL: %s", router.subscribeURL)
	}
}

func TestNewEventRouterRejectsUnsupportedScheme(t *testing.T) {
	_, err := NewEventRouter(
		"compose", 1, "file:///tmp/socket", "", "", &client.RancherClient{}, nil, "resource", 1, PingConfig{},
	)
	if err == nil {
		t.Fatal("expected unsupported event API URL scheme to fail")
	}
}

func TestEventSubscriptionRejectsCrossOriginAddress(t *testing.T) {
	_, err := validateEventSubscriptionURL(
		"https://api.example.test/v2-beta",
		"wss://metadata.example.test/v2-beta/subscribe",
	)
	if err == nil {
		t.Fatal("expected cross-origin event subscription URL to fail")
	}
}
