package integrations

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
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
	RetryPolicy string    `json:"retryPolicy,omitempty"`
	CheckedAt   time.Time `json:"checkedAt"`
}

type Options struct {
	JellyfinURL       string
	JellyfinKey       string
	PlexURL           string
	PlexToken         string
	EmbyURL           string
	EmbyKey           string
	PlexHistoryCursor string
	PriorRollups      []database.MediaActivityRollup
	Progress          func(Progress)
}

type Progress struct {
	TargetID        string `json:"targetId"`
	Phase           string `json:"phase"`
	Message         string `json:"message"`
	CurrentLabel    string `json:"currentLabel,omitempty"`
	Processed       int    `json:"processed"`
	Total           int    `json:"total"`
	ItemsImported   int    `json:"itemsImported"`
	RollupsImported int    `json:"rollupsImported"`
	UnmappedItems   int    `json:"unmappedItems"`
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
	retryPolicy := "3 attempts with Retry-After aware 429/5xx backoff"
	return []Target{
		{ID: "jellyfin", Name: "Jellyfin", Kind: "media_server", Status: checkServer(options.JellyfinURL, options.JellyfinKey, "jellyfin"), Description: "Sync metadata, artwork, collections, and library refresh events.", RetryPolicy: retryPolicy, CheckedAt: now},
		{ID: "plex", Name: "Plex", Kind: "media_server", Status: checkServer(options.PlexURL, options.PlexToken, "plex"), Description: "Sync metadata, artwork, collections, and library refresh events.", RetryPolicy: retryPolicy, CheckedAt: now},
		{ID: "emby", Name: "Emby", Kind: "media_server", Status: checkServer(options.EmbyURL, options.EmbyKey, "emby"), Description: "Sync metadata, artwork, collections, and library refresh events.", RetryPolicy: retryPolicy, CheckedAt: now},
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
	return syncEmbyFamily(ctx, options, "jellyfin", options.JellyfinURL, options.JellyfinKey, "Jellyfin", mappings)
}

func SyncEmby(ctx context.Context, options Options, mappings []database.PathMapping) (database.MediaServerSnapshot, error) {
	return syncEmbyFamily(ctx, options, "emby", options.EmbyURL, options.EmbyKey, "Emby", mappings)
}

func syncEmbyFamily(ctx context.Context, options Options, targetID string, rawBaseURL string, rawToken string, defaultName string, mappings []database.PathMapping) (database.MediaServerSnapshot, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(rawBaseURL), "/")
	token := strings.TrimSpace(rawToken)
	startedAt := time.Now().UTC()
	if baseURL == "" || token == "" {
		return database.MediaServerSnapshot{}, errors.New(targetID + " is not configured")
	}
	reportProgress(options, Progress{TargetID: targetID, Phase: "connecting", Message: "Connecting to " + defaultName})

	var info jellyfinSystemInfo
	if err := getJSON(ctx, baseURL+"/System/Info", "X-Emby-Token", token, &info); err != nil {
		return database.MediaServerSnapshot{}, err
	}
	serverName := strings.TrimSpace(info.ServerName)
	if serverName == "" {
		serverName = defaultName
	}

	var jellyfinUsers []jellyfinUser
	reportProgress(options, Progress{TargetID: targetID, Phase: "users", Message: "Reading " + defaultName + " users"})
	if err := getJSON(ctx, baseURL+"/Users", "X-Emby-Token", token, &jellyfinUsers); err != nil {
		return database.MediaServerSnapshot{}, err
	}
	users := make([]database.MediaServerUser, 0, len(jellyfinUsers))
	for _, user := range jellyfinUsers {
		if strings.TrimSpace(user.ID) == "" {
			continue
		}
		users = append(users, database.MediaServerUser{
			ServerID:    targetID,
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
		label := strings.TrimSpace(user.Name)
		if label == "" {
			label = "server"
		}
		reportProgress(options, Progress{TargetID: targetID, Phase: "items", Message: "Reading " + defaultName + " items for " + label, CurrentLabel: label})
		if err := syncEmbyFamilyItemsForUser(ctx, options, targetID, baseURL, token, user.ID, mappings, itemByID, fileByKey, rollupByItemID); err != nil {
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
	reportProgress(options, Progress{TargetID: targetID, Phase: "complete", Message: defaultName + " sync completed", ItemsImported: len(items), RollupsImported: len(rollups), UnmappedItems: unmapped})
	return database.MediaServerSnapshot{
		Server: database.MediaServer{
			ID:           targetID,
			Kind:         targetID,
			Name:         serverName,
			BaseURL:      baseURL,
			Status:       "configured",
			LastSyncedAt: completedAt,
			UpdatedAt:    completedAt,
		},
		Users: users,
		Libraries: []database.MediaServerLibrary{
			{ServerID: targetID, ExternalID: targetID + "-all", Name: defaultName + " Libraries", Kind: "mixed", ItemCount: len(items)},
		},
		Items:   items,
		Files:   files,
		Rollups: rollups,
		Job: database.MediaSyncJob{
			ID:              "sync_" + targetID + "_" + completedAt.Format("20060102150405"),
			ServerID:        targetID,
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
	reportProgress(options, Progress{TargetID: "plex", Phase: "connecting", Message: "Connecting to Plex"})

	var identity plexIdentity
	if err := getXML(ctx, baseURL+"/identity", "X-Plex-Token", token, &identity); err != nil {
		return database.MediaServerSnapshot{}, err
	}
	serverName := strings.TrimSpace(identity.FriendlyName)
	if serverName == "" {
		serverName = "Plex"
	}

	var sections plexSectionsResponse
	reportProgress(options, Progress{TargetID: "plex", Phase: "libraries", Message: "Reading Plex libraries"})
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
		reportProgress(options, Progress{TargetID: "plex", Phase: "items", Message: "Reading Plex library " + strings.TrimSpace(section.Title), CurrentLabel: strings.TrimSpace(section.Title)})
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
		for index, video := range library.Videos {
			externalID := firstNonEmpty(video.RatingKey, strings.TrimPrefix(video.Key, "/library/metadata/"))
			if externalID == "" {
				continue
			}
			reportProgress(options, Progress{TargetID: "plex", Phase: "items", Message: "Imported " + strings.TrimSpace(video.Title), CurrentLabel: strings.TrimSpace(video.Title), Processed: index + 1, Total: len(library.Videos), ItemsImported: len(itemByID) + 1})
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
	reportProgress(options, Progress{TargetID: "plex", Phase: "activity", Message: "Reading Plex playback history", ItemsImported: len(itemByID)})
	historyEndpoint := baseURL + "/status/sessions/history/all"
	cursorSeconds, _ := strconv.ParseInt(strings.TrimSpace(options.PlexHistoryCursor), 10, 64)
	if cursorSeconds > 0 {
		values := url.Values{}
		values.Set("viewedAt>", strconv.FormatInt(cursorSeconds, 10))
		historyEndpoint += "?" + values.Encode()
	}
	if err := getXML(ctx, historyEndpoint, "X-Plex-Token", token, &history); err != nil {
		return database.MediaServerSnapshot{}, err
	}
	rollupByItemID := map[string]*database.MediaActivityRollup{}
	baseUniqueUsers := map[string]int{}
	for _, prior := range options.PriorRollups {
		if prior.ServerID != "" && prior.ServerID != "plex" {
			continue
		}
		if strings.TrimSpace(prior.ItemExternalID) == "" {
			continue
		}
		priorCopy := prior
		priorCopy.ServerID = "plex"
		priorCopy.UpdatedAt = now
		rollupByItemID[prior.ItemExternalID] = &priorCopy
		baseUniqueUsers[prior.ItemExternalID] = prior.UniqueUsers
	}
	uniqueAccounts := map[string]map[string]bool{}
	maxViewedAt := cursorSeconds
	for _, event := range history.Videos {
		if strings.TrimSpace(event.RatingKey) == "" {
			continue
		}
		if cursorSeconds > 0 && event.ViewedAt <= cursorSeconds {
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
		if event.ViewedAt > maxViewedAt {
			maxViewedAt = event.ViewedAt
		}
	}
	rollups := make([]database.MediaActivityRollup, 0, len(rollupByItemID))
	for itemID, rollup := range rollupByItemID {
		if len(uniqueAccounts[itemID]) > 0 {
			rollup.UniqueUsers = baseUniqueUsers[itemID] + len(uniqueAccounts[itemID])
			rollup.WatchedUsers = rollup.UniqueUsers
		}
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
	reportProgress(options, Progress{TargetID: "plex", Phase: "complete", Message: "Plex sync completed", ItemsImported: len(items), RollupsImported: len(rollups), UnmappedItems: unmapped})
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
			Cursor:          strconv.FormatInt(maxViewedAt, 10),
			StartedAt:       startedAt,
			CompletedAt:     completedAt,
		},
	}, nil
}

func syncEmbyFamilyItemsForUser(ctx context.Context, options Options, targetID string, baseURL string, token string, userID string, mappings []database.PathMapping, itemByID map[string]database.MediaServerItem, fileByKey map[string]database.MediaServerFile, rollupByItemID map[string]*database.MediaActivityRollup) error {
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
		for index, sourceItem := range response.Items {
			if strings.TrimSpace(sourceItem.ID) == "" {
				continue
			}
			reportProgress(options, Progress{
				TargetID:        targetID,
				Phase:           "items",
				Message:         "Imported " + strings.TrimSpace(sourceItem.Name),
				CurrentLabel:    strings.TrimSpace(sourceItem.Name),
				Processed:       start + index + 1,
				Total:           response.TotalRecordCount,
				ItemsImported:   len(itemByID) + 1,
				RollupsImported: len(rollupByItemID),
			})
			if _, exists := itemByID[sourceItem.ID]; !exists {
				itemByID[sourceItem.ID] = database.MediaServerItem{
					ServerID:          targetID,
					ExternalID:        sourceItem.ID,
					LibraryExternalID: targetID + "-all",
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
				localPath, verification, confidence := applyPathMappings(targetID, path, mappings)
				fileByKey[sourceItem.ID+"|"+path] = database.MediaServerFile{
					ServerID:        targetID,
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
				localPath, verification, confidence := applyPathMappings(targetID, sourceItem.Path, mappings)
				fileByKey[sourceItem.ID+"|"+sourceItem.Path] = database.MediaServerFile{
					ServerID:        targetID,
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
					ServerID:       targetID,
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

func reportProgress(options Options, progress Progress) {
	if options.Progress == nil {
		return
	}
	progress.TargetID = strings.TrimSpace(progress.TargetID)
	progress.Phase = strings.TrimSpace(progress.Phase)
	progress.Message = strings.TrimSpace(progress.Message)
	progress.CurrentLabel = strings.TrimSpace(progress.CurrentLabel)
	if progress.Message == "" {
		progress.Message = progress.Phase
	}
	options.Progress(progress)
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
	body, err := providerBody(ctx, endpoint, headerName, token)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

func getXML(ctx context.Context, endpoint string, headerName string, token string, target any) error {
	body, err := providerBody(ctx, endpoint, headerName, token)
	if err != nil {
		return err
	}
	return xml.Unmarshal(body, target)
}

func providerBody(ctx context.Context, endpoint string, headerName string, token string) ([]byte, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	var lastStatus string
	for attempt := 1; attempt <= 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		if headerName != "" {
			req.Header.Set(headerName, token)
		}
		res, err := client.Do(req)
		if err != nil {
			if attempt == 3 || !sleepWithContext(ctx, retryDelay(attempt, "")) {
				return nil, err
			}
			continue
		}
		body, readErr := io.ReadAll(res.Body)
		_ = res.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		if res.StatusCode >= 200 && res.StatusCode < 300 {
			return body, nil
		}
		lastStatus = res.Status
		if res.StatusCode == http.StatusTooManyRequests || res.StatusCode >= 500 {
			if attempt < 3 && sleepWithContext(ctx, retryDelay(attempt, res.Header.Get("Retry-After"))) {
				continue
			}
		}
		return nil, errors.New("provider request failed with status " + res.Status)
	}
	if lastStatus == "" {
		lastStatus = "unknown"
	}
	return nil, errors.New("provider request failed with status " + lastStatus)
}

func retryDelay(attempt int, retryAfter string) time.Duration {
	retryAfter = strings.TrimSpace(retryAfter)
	if retryAfter != "" {
		if seconds, err := strconv.Atoi(retryAfter); err == nil && seconds >= 0 {
			if seconds > 5 {
				seconds = 5
			}
			return time.Duration(seconds) * time.Second
		}
		if retryAt, err := http.ParseTime(retryAfter); err == nil {
			delay := time.Until(retryAt)
			if delay > 0 && delay < 5*time.Second {
				return delay
			}
		}
	}
	switch attempt {
	case 1:
		return 250 * time.Millisecond
	case 2:
		return 750 * time.Millisecond
	default:
		return time.Second
	}
}

func sleepWithContext(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		return true
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
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
	return "", "server_reported", 0.68
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
