package database

import "testing"

func TestIntegrationSettingsPersistAndRedactSecrets(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	setting, err := store.UpsertIntegrationSetting(IntegrationSettingInput{
		Integration: "jellyfin",
		BaseURL:     "http://jellyfin.local:8096/",
		APIKey:      "jellyfin-secret-abcd",
	})
	if err != nil {
		t.Fatal(err)
	}
	if setting.BaseURL != "http://jellyfin.local:8096" || !setting.APIKeyConfigured || setting.APIKeyLast4 != "abcd" {
		t.Fatalf("setting = %#v", setting)
	}
	if setting.APIKey != "" {
		t.Fatal("redacted integration setting must not expose API key")
	}

	settings, err := store.ListIntegrationSettings()
	if err != nil {
		t.Fatal(err)
	}
	if len(settings) != 1 || settings[0].Integration != "jellyfin" || settings[0].APIKey != "" {
		t.Fatalf("settings = %#v", settings)
	}

	secrets, err := store.ListIntegrationSettingSecrets()
	if err != nil {
		t.Fatal(err)
	}
	if len(secrets) != 1 || secrets[0].APIKey != "jellyfin-secret-abcd" {
		t.Fatalf("secrets = %#v", secrets)
	}

	cleared, err := store.UpsertIntegrationSetting(IntegrationSettingInput{
		Integration:  "jellyfin",
		ClearAPIKey:  true,
		ClearBaseURL: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if cleared.BaseURL != "" || cleared.APIKeyConfigured {
		t.Fatalf("cleared = %#v", cleared)
	}
}

func TestIntegrationSettingsRejectUnknownIntegrations(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if _, err := store.UpsertIntegrationSetting(IntegrationSettingInput{Integration: "usenet", APIKey: "nope"}); err == nil {
		t.Fatal("expected unknown integration to be rejected")
	}
}
