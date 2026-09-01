package dto

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
)

func TestPackageUpdateJSON_PreservesExistingFields(t *testing.T) {
	update := PackageUpdate{
		Name:               "typescript",
		OldVersion:         "5.0.0",
		NewVersion:         "5.1.0",
		UpdateType:         manager.UpdateMinor,
		SizeBytes:          0,
		OldVersionPresence: PresenceObserved,
		NewVersionPresence: PresenceObserved,
		UpdateTypePresence: PresenceDerived,
		SizeBytesPresence:  PresenceUnavailable,
	}

	raw := marshalJSONMap(t, update)
	for _, key := range []string{"Name", "OldVersion", "NewVersion", "UpdateType", "SizeBytes"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("JSON missing existing field %q", key)
		}
	}
	for _, key := range []string{"OldVersionPresence", "NewVersionPresence", "UpdateTypePresence", "SizeBytesPresence"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("JSON missing additive presence field %q", key)
		}
	}
	if got := jsonString(t, raw["OldVersion"]); got != "5.0.0" {
		t.Errorf("OldVersion = %q, want observed 5.0.0", got)
	}
	if got := jsonString(t, raw["UpdateType"]); got != string(manager.UpdateMinor) {
		t.Errorf("UpdateType = %q, want derived minor", got)
	}
	if got := jsonString(t, raw["UpdateTypePresence"]); got != string(PresenceDerived) {
		t.Errorf("UpdateTypePresence = %q, want derived", got)
	}
	if got := jsonString(t, raw["SizeBytesPresence"]); got != string(PresenceUnavailable) {
		t.Errorf("SizeBytesPresence = %q, want unavailable", got)
	}
}

func TestPackageUpdateJSON_UnobservedIsNotFabricated(t *testing.T) {
	raw := marshalJSONMap(t, UnavailablePackageUpdate("git"))

	if got := jsonString(t, raw["OldVersion"]); got != "" {
		t.Errorf("OldVersion = %q, want empty unobserved value", got)
	}
	if got := jsonString(t, raw["NewVersion"]); got != "" {
		t.Errorf("NewVersion = %q, want empty unobserved value", got)
	}
	if got := jsonString(t, raw["UpdateType"]); got != "" {
		t.Errorf("UpdateType = %q, want empty unobserved value", got)
	}
	if number := jsonNumber(t, raw["SizeBytes"]); number != 0 {
		t.Errorf("SizeBytes = %v, want 0 with unavailable presence", number)
	}

	encoded, err := json.Marshal(UnavailablePackageUpdate("git"))
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	payload := string(encoded)
	for _, fabricated := range []string{`"unknown"`, `"minor"`} {
		if strings.Contains(payload, fabricated) {
			t.Errorf("JSON %s treats %s as an observed fact", payload, fabricated)
		}
	}
	if jsonString(t, raw["OldVersionPresence"]) != string(PresenceUnavailable) {
		t.Errorf("OldVersionPresence = %s, want unavailable", raw["OldVersionPresence"])
	}
}

func TestManagerUpdateResultJSON_PreservesExistingFields(t *testing.T) {
	result := ManagerUpdateResult{
		ID:                 manager.ManagerNPM,
		Name:               "NPM",
		Success:            true,
		UpdatedPackages:    []PackageUpdate{},
		SkippedPackages:    []string{},
		PackageCorrelation: CorrelationUnsupported,
		MetadataPilot:      true,
	}

	raw := marshalJSONMap(t, result)
	for _, key := range []string{"ID", "Name", "Success", "UpdatedPackages", "SkippedPackages", "Duration", "BytesDownloaded", "SpaceFreed"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("JSON missing existing field %q", key)
		}
	}
	if jsonString(t, raw["PackageCorrelation"]) != string(CorrelationUnsupported) {
		t.Errorf("PackageCorrelation = %s, want unsupported", raw["PackageCorrelation"])
	}
	if !jsonBool(t, raw["MetadataPilot"]) {
		t.Error("MetadataPilot = false, want true for npm")
	}
}

func marshalJSONMap(t *testing.T, value any) map[string]json.RawMessage {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	raw := map[string]json.RawMessage{}
	if err := json.Unmarshal(encoded, &raw); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	return raw
}

func jsonString(t *testing.T, raw json.RawMessage) string {
	t.Helper()
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("string field %s: %v", raw, err)
	}
	return value
}

func jsonNumber(t *testing.T, raw json.RawMessage) float64 {
	t.Helper()
	var value float64
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("number field %s: %v", raw, err)
	}
	return value
}

func jsonBool(t *testing.T, raw json.RawMessage) bool {
	t.Helper()
	var value bool
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("bool field %s: %v", raw, err)
	}
	return value
}
