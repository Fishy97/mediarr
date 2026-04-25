package integrations

import "time"

type Target struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Kind        string    `json:"kind"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	CheckedAt   time.Time `json:"checkedAt"`
}

func Defaults() []Target {
	now := time.Now().UTC()
	return []Target{
		{ID: "jellyfin", Name: "Jellyfin", Kind: "media_server", Status: "not_configured", Description: "Sync metadata, artwork, collections, and library refresh events.", CheckedAt: now},
		{ID: "plex", Name: "Plex", Kind: "media_server", Status: "not_configured", Description: "Sync metadata, artwork, collections, and library refresh events.", CheckedAt: now},
		{ID: "emby", Name: "Emby", Kind: "media_server", Status: "not_configured", Description: "Sync metadata, artwork, collections, and library refresh events.", CheckedAt: now},
		{ID: "ollama", Name: "Ollama", Kind: "local_ai", Status: "optional", Description: "Local-only advisory AI for matching, tags, and cleanup rationales.", CheckedAt: now},
	}
}
