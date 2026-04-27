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
	if !setting.AutoSyncEnabled || setting.AutoSyncIntervalMinutes != 360 {
		t.Fatalf("auto sync defaults = %#v", setting)
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
	if !settings[0].AutoSyncEnabled || settings[0].AutoSyncIntervalMinutes != 360 {
		t.Fatalf("redacted auto sync settings = %#v", settings[0])
	}

	secrets, err := store.ListIntegrationSettingSecrets()
	if err != nil {
		t.Fatal(err)
	}
	if len(secrets) != 1 || secrets[0].APIKey != "jellyfin-secret-abcd" {
		t.Fatalf("secrets = %#v", secrets)
	}

	disabled := false
	updated, err := store.UpsertIntegrationSetting(IntegrationSettingInput{
		Integration:             "jellyfin",
		AutoSyncEnabled:         &disabled,
		AutoSyncIntervalMinutes: 120,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.AutoSyncEnabled || updated.AutoSyncIntervalMinutes != 120 {
		t.Fatalf("updated auto sync settings = %#v", updated)
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
	if cleared.AutoSyncEnabled || cleared.AutoSyncIntervalMinutes != 120 {
		t.Fatalf("clearing credentials should preserve auto settings: %#v", cleared)
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
