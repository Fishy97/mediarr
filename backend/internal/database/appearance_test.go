package database

import "testing"

func TestAppearanceSettingsDefaultPersistAndValidate(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	defaults, err := store.GetAppearanceSettings()
	if err != nil {
		t.Fatal(err)
	}
	if defaults.Theme != "system" || defaults.CustomCSS != "" {
		t.Fatalf("default appearance = %#v", defaults)
	}

	updated, err := store.UpdateAppearanceSettings(AppearanceSettingsInput{
		Theme:     "light",
		CustomCSS: ".app-shell { letter-spacing: 0; }",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Theme != "light" || updated.CustomCSS != ".app-shell { letter-spacing: 0; }" || updated.UpdatedAt.IsZero() {
		t.Fatalf("updated appearance = %#v", updated)
	}

	loaded, err := store.GetAppearanceSettings()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Theme != updated.Theme || loaded.CustomCSS != updated.CustomCSS {
		t.Fatalf("loaded appearance = %#v, want %#v", loaded, updated)
	}

	if _, err := store.UpdateAppearanceSettings(AppearanceSettingsInput{Theme: "neon"}); err == nil {
		t.Fatal("invalid theme should fail")
	}
	if _, err := store.UpdateAppearanceSettings(AppearanceSettingsInput{Theme: "dark", CustomCSS: "@import url('https://example.test/theme.css');"}); err == nil {
		t.Fatal("custom css with network imports should fail")
	}
	if _, err := store.UpdateAppearanceSettings(AppearanceSettingsInput{Theme: "dark", CustomCSS: "body { background: url ('https://example.test/theme.css'); }"}); err == nil {
		t.Fatal("custom css with spaced url functions should fail")
	}
}
