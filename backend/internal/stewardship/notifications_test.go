package stewardship

import "testing"

func TestValidateWebhookURLAllowsHTTPAndHTTPSOnly(t *testing.T) {
	if err := ValidateWebhookURL("https://example.test/hook"); err != nil {
		t.Fatalf("https webhook rejected: %v", err)
	}
	if err := ValidateWebhookURL("ftp://example.test/hook"); err == nil {
		t.Fatal("expected ftp webhook to be rejected")
	}
}

func TestNotificationWithDefaultsIsUnreadAndInfoByDefault(t *testing.T) {
	notification := Notification{Title: "Campaign completed"}.WithDefaults()
	if notification.ID == "" || notification.Level != "info" || notification.Read {
		t.Fatalf("unexpected defaults: %#v", notification)
	}
	if notification.CreatedAt.IsZero() {
		t.Fatalf("CreatedAt not set: %#v", notification)
	}
}
