package handlers

import (
	"encoding/base64"
	"errors"
	"regexp"
	"strings"

	"github.com/PastureStack/compose-cli/internal/events"
	client "github.com/PastureStack/compose-cli/internal/rancherclient/v2"
)

var catalogPlanID = regexp.MustCompile(`^aicataloglaunch-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var catalogProofID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var errCatalogDelegation = errors.New("CatalogLaunchDelegationInvalid")

func catalogLaunchProof(event *events.Event) (string, bool, error) {
	if event == nil {
		return "", true, errCatalogDelegation
	}
	value, marked := event.Data["catalogLaunchDelegation"]
	if !marked {
		return "", false, nil
	}
	proof, ok := value.(map[string]interface{})
	if !ok || len(proof) != 4 {
		return "", true, errCatalogDelegation
	}
	plan, planOK := proof["planId"].(string)
	stack, stackOK := proof["stackId"].(string)
	id, idOK := proof["eventId"].(string)
	token, tokenOK := proof["proof"].(string)
	if !planOK || !stackOK || !idOK || !tokenOK || !catalogPlanID.MatchString(plan) || stack == "" || len(stack) > 128 || stack != event.ResourceID || id == "" || len(id) > 128 || id != event.ID {
		return "", true, errCatalogDelegation
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != "cld1" || !catalogProofID.MatchString(parts[1]) || len(parts[2]) != 43 {
		return "", true, errCatalogDelegation
	}
	nonce, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(nonce) != 32 {
		return "", true, errCatalogDelegation
	}
	return token, true, nil
}

func catalogLaunchClient(event *events.Event, original *client.RancherClient) (*client.RancherClient, error) {
	proof, marked, err := catalogLaunchProof(event)
	if err != nil {
		return nil, err
	}
	if !marked {
		return original, nil
	}
	return client.WithCatalogLaunchDelegation(original, proof)
}
