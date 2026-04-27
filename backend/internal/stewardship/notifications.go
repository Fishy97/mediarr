package stewardship

import (
	"errors"
	"net/url"
	"strings"
)

func ValidateWebhookURL(rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return errors.New("webhook url is required")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
	default:
		return errors.New("webhook url must use http or https")
	}
	if parsed.Host == "" {
		return errors.New("webhook url host is required")
	}
	return nil
}
