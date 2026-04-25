package integrations

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Fishy97/mediarr/backend/internal/database"
)

type Target struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Kind        string    `json:"kind"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	CheckedAt   time.Time `json:"checkedAt"`
}

type Options struct {
	JellyfinURL string
	JellyfinKey string
	PlexURL     string
	PlexToken   string
	EmbyURL     string
	EmbyKey     string
}

type RefreshResult struct {
	TargetID    string    `json:"targetId"`
	Status      string    `json:"status"`
	Message     string    `json:"message"`
	RequestedAt time.Time `json:"requestedAt"`
}

type jellyfinSystemInfo struct {
	ServerName string `json:"ServerName"`
}

type jellyfinUser struct {
	ID   string `json:"Id"`
	Name string `json:"Name"`
}

type jellyfinItemsResponse struct {
	Items            []jellyfinItem `json:"Items"`
	TotalRecordCount int            `json:"TotalRecordCount"`
}

type jellyfinItem struct {
	ID             string                `json:"Id"`
	Name           string                `json:"Name"`
	Type           string                `json:"Type"`
	ProductionYear int                   `json:"ProductionYear"`
	Path           string                `json:"Path"`
	ProviderIDs    map[string]string     `json:"ProviderIds"`
	RunTimeTicks   int64                 `json:"RunTimeTicks"`
	DateCreated    string                `json:"DateCreated"`
	MediaSources   []jellyfinMediaSource `json:"MediaSources"`
	UserData       jellyfinUserData      `json:"UserData"`
	ParentID       string                `json:"ParentId"`
	SeriesID       string                `json:"SeriesId"`
}

type jellyfinMediaSource struct {
	Path      string `json:"Path"`
	Size      int64  `json:"Size"`
	Container string `json:"Container"`
}

type jellyfinUserData struct {
	PlayCount      int    `json:"PlayCount"`
	LastPlayedDate string `json:"LastPlayedDate"`
	Played         bool   `json:"Played"`
	IsFavorite     bool   `json:"IsFavorite"`
}

type plexIdentity struct {
	FriendlyName string `xml:"friendlyName,attr"`
}

type plexSectionsResponse struct {
	Directories []plexDirectory `xml:"Directory"`
}

type plexDirectory struct {
	Key   string `xml:"key,attr"`
	Title string `xml:"title,attr"`
	Type  string `xml:"type,attr"`
}

type plexLibraryResponse struct {
	Videos []plexVideo `xml:"Video"`
}

type plexVideo struct {
	RatingKey string      `xml:"ratingKey,attr"`
	Key       string      `xml:"key,attr"`
	Title     string      `xml:"title,attr"`
	Type      string      `xml:"type,attr"`
	Year      int         `xml:"year,attr"`
	AddedAt   int64       `xml:"addedAt,attr"`
	Duration  int         `xml:"duration,attr"`
	Guids     []plexGuid  `xml:"Guid"`
	Media     []plexMedia `xml:"Media"`
}

type plexGuid struct {
	ID string `xml:"id,attr"`
}

type plexMedia struct {
	Duration int        `xml:"duration,attr"`
	Parts    []plexPart `xml:"Part"`
}

type plexPart struct {
	File      string `xml:"file,attr"`
	Size      int64  `xml:"size,attr"`
	Container string `xml:"container,attr"`
}

type plexHistoryResponse struct {
	Videos []plexHistoryVideo `xml:"Video"`
}

type plexHistoryVideo struct {
	RatingKey string `xml:"ratingKey,attr"`
	ViewedAt  int64  `xml:"viewedAt,attr"`
	AccountID string `xml:"accountID,attr"`
}

func Defaults() []Target {
	return DefaultsWithOptions(Options{})
}

func DefaultsWithOptions(options Options) []Target {
	now := time.Now().UTC()
	return []Target{
		{ID: "jellyfin", Name: "Jellyfin", Kind: "media_server", Status: checkServer(options.JellyfinURL, options.JellyfinKey, "jellyfin"), Description: "Sync metadata, artwork, collections, and library refresh events.", CheckedAt: now},
		{ID: "plex", Name: "Plex", Kind: "media_server", Status: checkServer(options.PlexURL, options.PlexToken, "plex"), Description: "Sync metadata, artwork, collections, and library refresh events.", CheckedAt: now},
		{ID: "emby", Name: "Emby", Kind: "media_server", Status: checkServer(options.EmbyURL, options.EmbyKey, "emby"), Description: "Sync metadata, artwork, collections, and library refresh events.", CheckedAt: now},
		{ID: "ollama", Name: "Ollama", Kind: "local_ai", Status: "optional", Description: "Local-only advisory AI for matching, tags, and cleanup rationales.", CheckedAt: now},
	}
}

func Refresh(ctx context.Context, options Options, targetID string) (RefreshResult, error) {
	targetID = strings.ToLower(strings.TrimSpace(targetID))
	requestedAt := time.Now().UTC()
	endpoint, token, method, headerName, err := refreshConfig(options, targetID)
	if err != nil {
		return RefreshResult{TargetID: targetID, Status: "failed", Message: err.Error(), RequestedAt: requestedAt}, err
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, nil)
	if err != nil {
		return RefreshResult{TargetID: targetID, Status: "failed", Message: "invalid refresh endpoint", RequestedAt: requestedAt}, err
	}
	if headerName != "" {
		req.Header.Set(headerName, token)
	}
	res, err := (&http.Client{Timeout: 8 * time.Second}).Do(req)
	if err != nil {
		return RefreshResult{TargetID: targetID, Status: "failed", Message: err.Error(), RequestedAt: requestedAt}, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		message := "refresh failed with status " + res.Status
		return RefreshResult{TargetID: targetID, Status: "failed", Message: message, RequestedAt: requestedAt}, errors.New(message)
	}
	return RefreshResult{TargetID: targetID, Status: "requested", Message: "library refresh requested", RequestedAt: requestedAt}, nil
}

func SyncJellyfin(ctx context.Context, options Options, mappings []database.PathMapping) (database.MediaServerSnapshot, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(options.JellyfinURL), "/")
	token := strings.TrimSpace(options.JellyfinKey)
	startedAt := time.Now().UTC()
	if baseURL == "" || token == "" {
		return database.MediaServerSnapshot{}, errors.New("jellyfin is not configured")
	}

	var info jellyfinSystemInfo
	if err := getJSON(ctx, baseURL+"/System/Info", "X-Emby-Token", token, &info); err != nil {
		return database.MediaServerSnapshot{}, err
	}
	serverName := strings.TrimSpace(info.ServerName)
	if serverName == "" {
		serverName = "Jellyfin"
	}

	var jellyfinUsers []jellyfinUser
	if err := getJSON(ctx, baseURL+"/Users", "X-Emby-Token", token, &jellyfinUsers); err != nil {
		return database.MediaServerSnapshot{}, err
	}
	users := make([]database.MediaServerUser, 0, len(jellyfinUsers))
	for _, user := range jellyfinUsers {
		if strings.TrimSpace(user.ID) == "" {
			continue
		}
		users = append(users, database.MediaServerUser{
			ServerID:    "jellyfin",
			ExternalID:  user.ID,
			DisplayName: strings.TrimSpace(user.Name),
		})
	}
	if len(jellyfinUsers) == 0 {
		jellyfinUsers = []jellyfinUser{{ID: "", Name: "Server"}}
	}

	itemByID := map[string]database.MediaServerItem{}
	fileByKey := map[string]database.MediaServerFile{}
	rollupByItemID := map[string]*database.MediaActivityRollup{}
	for _, user := range jellyfinUsers {
		if err := syncJellyfinItemsForUser(ctx, baseURL, token, user.ID, mappings, itemByID, fileByKey, rollupByItemID); err != nil {
			return database.MediaServerSnapshot{}, err
		}
	}

	items := make([]database.MediaServerItem, 0, len(itemByID))
	for _, item := range itemByID {
		items = append(items, item)
	}
	files := make([]database.MediaServerFile, 0, len(fileByKey))
	unmapped := 0
	for _, file := range fileByKey {
		if file.LocalPath == "" {
			unmapped++
		}
		files = append(files, file)
	}
	rollups := make([]database.MediaActivityRollup, 0, len(rollupByItemID))
	for _, rollup := range rollupByItemID {
		rollups = append(rollups, *rollup)
	}
	completedAt := time.Now().UTC()
	return database.MediaServerSnapshot{
		Server: database.MediaServer{
			ID:           "jellyfin",
			Kind:         "jellyfin",
			Name:         serverName,
			BaseURL:      baseURL,
			Status:       "configured",
			LastSyncedAt: completedAt,
			UpdatedAt:    completedAt,
		},
		Users: users,
		Libraries: []database.MediaServerLibrary{
			{ServerID: "jellyfin", ExternalID: "jellyfin-all", Name: "Jellyfin Libraries", Kind: "mixed", ItemCount: len(items)},
		},
		Items:   items,
		Files:   files,
		Rollups: rollups,
		Job: database.MediaSyncJob{
			ID:              "sync_jellyfin_" + completedAt.Format("20060102150405"),
			ServerID:        "jellyfin",
			Status:          "completed",
			ItemsImported:   len(items),
			RollupsImported: len(rollups),
			UnmappedItems:   unmapped,
			StartedAt:       startedAt,
			CompletedAt:     completedAt,
		},
	}, nil
}

func SyncPlex(ctx context.Context, options Options, mappings []database.PathMapping) (database.MediaServerSnapshot, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(options.PlexURL), "/")
	token := strings.TrimSpace(options.PlexToken)
	startedAt := time.Now().UTC()
	if baseURL == "" || token == "" {
		return database.MediaServerSnapshot{}, errors.New("plex is not configured")
	}

	var identity plexIdentity
	if err := getXML(ctx, baseURL+"/identity", "X-Plex-Token", token, &identity); err != nil {
		return database.MediaServerSnapshot{}, err
	}
	serverName := strings.TrimSpace(identity.FriendlyName)
	if serverName == "" {
		serverName = "Plex"
	}

	var sections plexSectionsResponse
	if err := getXML(ctx, baseURL+"/library/sections", "X-Plex-Token", token, &sections); err != nil {
		return database.MediaServerSnapshot{}, err
	}

	now := time.Now().UTC()
	var libraries []database.MediaServerLibrary
	itemByID := map[string]database.MediaServerItem{}
	fileByKey := map[string]database.MediaServerFile{}
	for _, section := range sections.Directories {
		if strings.TrimSpace(section.Key) == "" {
			continue
		}
		libraryKind := normalizePlexKind(section.Type)
		var library plexLibraryResponse
		if err := getXML(ctx, baseURL+"/library/sections/"+url.PathEscape(section.Key)+"/all", "X-Plex-Token", token, &library); err != nil {
			return database.MediaServerSnapshot{}, err
		}
		libraries = append(libraries, database.MediaServerLibrary{
			ServerID:   "plex",
			ExternalID: section.Key,
			Name:       strings.TrimSpace(section.Title),
			Kind:       libraryKind,
			ItemCount:  len(library.Videos),
		})
		for _, video := range library.Videos {
			externalID := firstNonEmpty(video.RatingKey, strings.TrimPrefix(video.Key, "/library/metadata/"))
			if externalID == "" {
				continue
			}
			runtime := video.Duration / 1000
			if runtime == 0 && len(video.Media) > 0 {
				runtime = video.Media[0].Duration / 1000
			}
			itemByID[externalID] = database.MediaServerItem{
				ServerID:          "plex",
				ExternalID:        externalID,
				LibraryExternalID: section.Key,
				Kind:              normalizePlexKind(firstNonEmpty(video.Type, section.Type)),
				Title:             strings.TrimSpace(video.Title),
				Year:              video.Year,
				ProviderIDs:       plexProviderIDs(video.Guids),
				RuntimeSeconds:    runtime,
				DateCreated:       unixTime(video.AddedAt),
				MatchConfidence:   0.78,
				UpdatedAt:         now,
			}
			for _, media := range video.Media {
				for _, part := range media.Parts {
					if strings.TrimSpace(part.File) == "" {
						continue
					}
					localPath, verification, confidence := applyPathMappings("plex", part.File, mappings)
					fileByKey[externalID+"|"+part.File] = database.MediaServerFile{
						ServerID:        "plex",
						ItemExternalID:  externalID,
						Path:            part.File,
						SizeBytes:       part.Size,
						Container:       part.Container,
						LocalPath:       localPath,
						Verification:    verification,
						MatchConfidence: confidence,
					}
				}
			}
		}
	}

	var history plexHistoryResponse
	if err := getXML(ctx, baseURL+"/status/sessions/history/all", "X-Plex-Token", token, &history); err != nil {
		return database.MediaServerSnapshot{}, err
	}
	rollupByItemID := map[string]*database.MediaActivityRollup{}
	uniqueAccounts := map[string]map[string]bool{}
	for _, event := range history.Videos {
		if strings.TrimSpace(event.RatingKey) == "" {
			continue
		}
		rollup := rollupByItemID[event.RatingKey]
		if rollup == nil {
			rollup = &database.MediaActivityRollup{ServerID: "plex", ItemExternalID: event.RatingKey, UpdatedAt: now}
			rollupByItemID[event.RatingKey] = rollup
		}
		rollup.PlayCount++
		viewedAt := unixTime(event.ViewedAt)
		if !viewedAt.IsZero() && (rollup.LastPlayedAt.IsZero() || viewedAt.After(rollup.LastPlayedAt)) {
			rollup.LastPlayedAt = viewedAt
		}
		accountID := strings.TrimSpace(event.AccountID)
		if accountID != "" {
			if uniqueAccounts[event.RatingKey] == nil {
				uniqueAccounts[event.RatingKey] = map[string]bool{}
			}
			uniqueAccounts[event.RatingKey][accountID] = true
		}
	}
	rollups := make([]database.MediaActivityRollup, 0, len(rollupByItemID))
	for itemID, rollup := range rollupByItemID {
		rollup.UniqueUsers = len(uniqueAccounts[itemID])
		rollup.WatchedUsers = rollup.UniqueUsers
		if rollup.UniqueUsers == 0 && rollup.PlayCount > 0 {
			rollup.UniqueUsers = 1
			rollup.WatchedUsers = 1
		}
		rollups = append(rollups, *rollup)
	}

	items := make([]database.MediaServerItem, 0, len(itemByID))
	for _, item := range itemByID {
		items = append(items, item)
	}
	files := make([]database.MediaServerFile, 0, len(fileByKey))
	unmapped := 0
	for _, file := range fileByKey {
		if file.LocalPath == "" {
			unmapped++
		}
		files = append(files, file)
	}
	completedAt := time.Now().UTC()
	return database.MediaServerSnapshot{
		Server: database.MediaServer{
			ID:           "plex",
			Kind:         "plex",
			Name:         serverName,
			BaseURL:      baseURL,
			Status:       "configured",
			LastSyncedAt: completedAt,
			UpdatedAt:    completedAt,
		},
		Libraries: libraries,
		Items:     items,
		Files:     files,
		Rollups:   rollups,
		Job: database.MediaSyncJob{
			ID:              "sync_plex_" + completedAt.Format("20060102150405"),
			ServerID:        "plex",
			Status:          "completed",
			ItemsImported:   len(items),
			RollupsImported: len(rollups),
			UnmappedItems:   unmapped,
			StartedAt:       startedAt,
			CompletedAt:     completedAt,
		},
	}, nil
}

func syncJellyfinItemsForUser(ctx context.Context, baseURL string, token string, userID string, mappings []database.PathMapping, itemByID map[string]database.MediaServerItem, fileByKey map[string]database.MediaServerFile, rollupByItemID map[string]*database.MediaActivityRollup) error {
	const limit = 200
	for start := 0; ; start += limit {
		values := url.Values{}
		values.Set("Recursive", "true")
		values.Set("StartIndex", intString(start))
		values.Set("Limit", intString(limit))
		values.Set("IncludeItemTypes", "Movie,Series,Season,Episode,Video")
		values.Set("Fields", "Path,ProviderIds,MediaSources,DateCreated,UserData,RunTimeTicks")
		values.Set("EnableUserData", "true")
		if strings.TrimSpace(userID) != "" {
			values.Set("userId", userID)
		}
		var response jellyfinItemsResponse
		if err := getJSON(ctx, baseURL+"/Items?"+values.Encode(), "X-Emby-Token", token, &response); err != nil {
			return err
		}
		now := time.Now().UTC()
		for _, sourceItem := range response.Items {
			if strings.TrimSpace(sourceItem.ID) == "" {
				continue
			}
			if _, exists := itemByID[sourceItem.ID]; !exists {
				itemByID[sourceItem.ID] = database.MediaServerItem{
					ServerID:          "jellyfin",
					ExternalID:        sourceItem.ID,
					LibraryExternalID: "jellyfin-all",
					ParentExternalID:  firstNonEmpty(sourceItem.SeriesID, sourceItem.ParentID),
					Kind:              normalizeJellyfinKind(sourceItem.Type),
					Title:             strings.TrimSpace(sourceItem.Name),
					Year:              sourceItem.ProductionYear,
					Path:              strings.TrimSpace(sourceItem.Path),
					ProviderIDs:       sourceItem.ProviderIDs,
					RuntimeSeconds:    int(sourceItem.RunTimeTicks / 10_000_000),
					DateCreated:       parseProviderTime(sourceItem.DateCreated),
					MatchConfidence:   0.8,
					UpdatedAt:         now,
				}
			}
			for _, mediaSource := range sourceItem.MediaSources {
				path := firstNonEmpty(mediaSource.Path, sourceItem.Path)
				if strings.TrimSpace(path) == "" {
					continue
				}
				localPath, verification, confidence := applyPathMappings("jellyfin", path, mappings)
				fileByKey[sourceItem.ID+"|"+path] = database.MediaServerFile{
					ServerID:        "jellyfin",
					ItemExternalID:  sourceItem.ID,
					Path:            path,
					SizeBytes:       mediaSource.Size,
					Container:       mediaSource.Container,
					LocalPath:       localPath,
					Verification:    verification,
					MatchConfidence: confidence,
				}
			}
			if len(sourceItem.MediaSources) == 0 && strings.TrimSpace(sourceItem.Path) != "" {
				localPath, verification, confidence := applyPathMappings("jellyfin", sourceItem.Path, mappings)
				fileByKey[sourceItem.ID+"|"+sourceItem.Path] = database.MediaServerFile{
					ServerID:        "jellyfin",
					ItemExternalID:  sourceItem.ID,
					Path:            sourceItem.Path,
					LocalPath:       localPath,
					Verification:    verification,
					MatchConfidence: confidence,
				}
			}
			rollup := rollupByItemID[sourceItem.ID]
			if rollup == nil {
				rollup = &database.MediaActivityRollup{
					ServerID:       "jellyfin",
					ItemExternalID: sourceItem.ID,
					UpdatedAt:      now,
				}
				rollupByItemID[sourceItem.ID] = rollup
			}
			rollup.PlayCount += sourceItem.UserData.PlayCount
			lastPlayed := parseProviderTime(sourceItem.UserData.LastPlayedDate)
			if !lastPlayed.IsZero() && (rollup.LastPlayedAt.IsZero() || lastPlayed.After(rollup.LastPlayedAt)) {
				rollup.LastPlayedAt = lastPlayed
			}
			if sourceItem.UserData.PlayCount > 0 || sourceItem.UserData.Played || !lastPlayed.IsZero() {
				rollup.UniqueUsers++
				rollup.WatchedUsers++
			}
			if sourceItem.UserData.IsFavorite {
				rollup.FavoriteCount++
			}
		}
		if len(response.Items) < limit || (response.TotalRecordCount > 0 && start+len(response.Items) >= response.TotalRecordCount) {
			break
		}
	}
	return nil
}

func refreshConfig(options Options, targetID string) (endpoint string, token string, method string, headerName string, err error) {
	switch targetID {
	case "jellyfin":
		baseURL := strings.TrimRight(strings.TrimSpace(options.JellyfinURL), "/")
		token := strings.TrimSpace(options.JellyfinKey)
		if baseURL == "" || token == "" {
			return "", "", "", "", errors.New("jellyfin is not configured")
		}
		return baseURL + "/Library/Refresh", token, http.MethodPost, "X-Emby-Token", nil
	case "emby":
		baseURL := strings.TrimRight(strings.TrimSpace(options.EmbyURL), "/")
		token := strings.TrimSpace(options.EmbyKey)
		if baseURL == "" || token == "" {
			return "", "", "", "", errors.New("emby is not configured")
		}
		return baseURL + "/Library/Refresh", token, http.MethodPost, "X-Emby-Token", nil
	case "plex":
		baseURL := strings.TrimRight(strings.TrimSpace(options.PlexURL), "/")
		token := strings.TrimSpace(options.PlexToken)
		if baseURL == "" || token == "" {
			return "", "", "", "", errors.New("plex is not configured")
		}
		values := url.Values{}
		values.Set("X-Plex-Token", token)
		return baseURL + "/library/sections/all/refresh?" + values.Encode(), token, http.MethodGet, "", nil
	default:
		return "", "", "", "", errors.New("unknown integration target")
	}
}

func checkServer(baseURL string, token string, kind string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	token = strings.TrimSpace(token)
	if baseURL == "" || token == "" {
		return "not_configured"
	}
	path := "/System/Info"
	if kind == "plex" {
		path = "/identity?X-Plex-Token=" + token
	}
	req, err := http.NewRequest(http.MethodGet, baseURL+path, nil)
	if err != nil {
		return "invalid_config"
	}
	if kind != "plex" {
		req.Header.Set("X-Emby-Token", token)
	}
	client := &http.Client{Timeout: 4 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return "unavailable"
	}
	defer res.Body.Close()
	switch {
	case res.StatusCode >= 200 && res.StatusCode < 300:
		return "configured"
	case res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden:
		return "invalid_credentials"
	default:
		return "degraded"
	}
}

func getJSON(ctx context.Context, endpoint string, headerName string, token string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	if headerName != "" {
		req.Header.Set(headerName, token)
	}
	res, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return errors.New("provider request failed with status " + res.Status)
	}
	return json.NewDecoder(res.Body).Decode(target)
}

func getXML(ctx context.Context, endpoint string, headerName string, token string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	if headerName != "" {
		req.Header.Set(headerName, token)
	}
	res, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return errors.New("provider request failed with status " + res.Status)
	}
	return xml.NewDecoder(res.Body).Decode(target)
}

func intString(value int) string {
	return strconv.Itoa(value)
}

func normalizeJellyfinKind(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "movie":
		return "movie"
	case "series":
		return "series"
	case "season":
		return "season"
	case "episode":
		return "episode"
	default:
		return "video"
	}
}

func normalizePlexKind(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "movie":
		return "movie"
	case "show":
		return "series"
	case "season":
		return "season"
	case "episode":
		return "episode"
	default:
		return "video"
	}
}

func plexProviderIDs(guids []plexGuid) map[string]string {
	providerIDs := map[string]string{}
	for _, guid := range guids {
		provider, id, ok := strings.Cut(strings.TrimSpace(guid.ID), "://")
		if !ok || provider == "" || id == "" {
			continue
		}
		providerIDs[strings.ToLower(provider)] = id
	}
	return providerIDs
}

func applyPathMappings(serverID string, path string, mappings []database.PathMapping) (string, string, float64) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", "server_reported", 0.4
	}
	if path == "/media" || strings.HasPrefix(path, "/media/") {
		return path, "local_verified", 0.92
	}
	for _, mapping := range mappings {
		if mapping.ServerID != "" && mapping.ServerID != serverID {
			continue
		}
		serverPrefix := strings.TrimRight(strings.TrimSpace(mapping.ServerPathPrefix), "/")
		localPrefix := strings.TrimRight(strings.TrimSpace(mapping.LocalPathPrefix), "/")
		if serverPrefix == "" || localPrefix == "" {
			continue
		}
		if path == serverPrefix {
			return localPrefix, "path_mapped", 0.86
		}
		if strings.HasPrefix(path, serverPrefix+"/") {
			return localPrefix + strings.TrimPrefix(path, serverPrefix), "path_mapped", 0.86
		}
	}
	return "", "server_reported", 0.62
}

func unixTime(seconds int64) time.Time {
	if seconds <= 0 {
		return time.Time{}
	}
	return time.Unix(seconds, 0).UTC()
}

func parseProviderTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.0000000Z", "2006-01-02T15:04:05Z"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed.UTC()
		}
	}
	return time.Time{}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
