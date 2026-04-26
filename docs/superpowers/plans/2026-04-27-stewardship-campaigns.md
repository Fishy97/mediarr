# Stewardship Campaigns Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add production-grade stewardship campaigns: saved rule sets that can be simulated, audited, and used to generate non-destructive Mediarr recommendations.

**Architecture:** Add a focused `backend/internal/campaigns` domain package for campaign rules and simulation. Persist campaign definitions and runs through SQLite in `backend/internal/database`, expose REST endpoints in `backend/internal/api`, and add a compact Campaigns view in the React UI. Campaign runs create ordinary suggest-only recommendations with campaign evidence attached and never mutate media files.

**Tech Stack:** Go, SQLite, existing REST API, React/TypeScript, Vitest, Docker Compose, existing no-delete invariant.

---

## File Structure

- Create `backend/internal/campaigns/types.go`: campaign, rule, candidate, simulation result, and run summary types.
- Create `backend/internal/campaigns/engine.go`: rule evaluation, suppression, simulation, and recommendation conversion.
- Create `backend/internal/campaigns/engine_test.go`: operator, simulation, suppression, and recommendation evidence tests.
- Modify `backend/internal/recommendations/engine.go`: add `ActionReviewCampaignMatch`.
- Modify `backend/internal/database/store.go`: schema, campaign CRUD, run persistence, and campaign recommendation replacement.
- Create `backend/internal/database/campaigns_test.go`: persistence and replacement tests.
- Modify `backend/internal/api/server.go`: campaign REST routes and handlers.
- Create `backend/internal/api/campaigns_test.go`: API CRUD, simulate, run, and no-delete tests.
- Modify `frontend/src/types.ts`: campaign types.
- Modify `frontend/src/lib/api.ts`: campaign API helpers.
- Modify `frontend/src/lib/api.test.ts`: endpoint tests.
- Modify `frontend/src/App.tsx`: add Campaigns nav/view, editor, simulation, and run controls.
- Modify `frontend/src/styles/app.css`: campaign view styles.
- Modify `README.md`, `docs/api/rest.md`, and `docs/recommendation-proof.md`: campaign docs.

## Task 1: Campaign Domain Engine

**Files:**
- Create: `backend/internal/campaigns/types.go`
- Create: `backend/internal/campaigns/engine.go`
- Create: `backend/internal/campaigns/engine_test.go`
- Modify: `backend/internal/recommendations/engine.go`

- [ ] **Step 1: Write failing operator tests**

Add tests in `backend/internal/campaigns/engine_test.go`:

```go
func TestRuleOperatorsEvaluateCandidates(t *testing.T) {
	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)
	candidate := Candidate{
		Key: "jellyfin:item_1",
		Title: "Arrival",
		Kind: "movie",
		LibraryName: "Movies",
		Verification: "local_verified",
		EstimatedSavingsBytes: 42_000_000_000,
		VerifiedSavingsBytes: 42_000_000_000,
		Confidence: 0.92,
		AddedAt: now.AddDate(-1, 0, 0),
		LastPlayedAt: now.AddDate(-2, 0, 0),
		PlayCount: 3,
		UniqueUsers: 1,
		FavoriteCount: 0,
		AffectedPaths: []string{"/media/movies/Arrival.mkv"},
	}
	tests := []struct {
		name string
		rule Rule
		want bool
	}{
		{name: "equals", rule: Rule{Field: FieldKind, Operator: OperatorEquals, Value: "movie"}, want: true},
		{name: "in", rule: Rule{Field: FieldVerification, Operator: OperatorIn, Values: []string{"path_mapped", "local_verified"}}, want: true},
		{name: "greater equal", rule: Rule{Field: FieldEstimatedSavingsBytes, Operator: OperatorGreaterOrEqual, Value: "40000000000"}, want: true},
		{name: "less equal days", rule: Rule{Field: FieldLastPlayedDays, Operator: OperatorGreaterOrEqual, Value: "540"}, want: true},
		{name: "is not empty", rule: Rule{Field: FieldLastPlayedDays, Operator: OperatorIsNotEmpty}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluateRule(tt.rule, candidate, now)
			if got.Matched != tt.want {
				t.Fatalf("matched = %v, want %v; result=%#v", got.Matched, tt.want, got)
			}
		})
	}
}
```

Run:

```bash
cd /Users/mat/Desktop/media-library-manager/backend
go test ./internal/campaigns -run TestRuleOperatorsEvaluateCandidates -count=1
```

Expected: fail because the package does not exist.

- [ ] **Step 2: Implement campaign types and operators**

Create `types.go` with constants for fields/operators and structs. Implement `EvaluateRule` in `engine.go` with typed parsing for string, number, and day-derived fields.

- [ ] **Step 3: Write failing simulation/recommendation tests**

Add tests proving:

- disabled campaigns still simulate but mark `Enabled=false`
- favorite candidates are suppressed
- low-confidence candidates are suppressed
- campaign recommendations are `destructive=false`
- evidence includes `campaignId`, `campaignName`, and `matchedRules`

Run:

```bash
cd /Users/mat/Desktop/media-library-manager/backend
go test ./internal/campaigns -count=1
```

- [ ] **Step 4: Implement simulation and recommendation conversion**

Implement:

- `Simulate(campaign Campaign, candidates []Candidate, now time.Time) Result`
- `RecommendationForMatch(campaign Campaign, runID string, item ResultItem) recommendations.Recommendation`
- suppression reasons for favorite, low confidence, missing paths, and below storage threshold.

- [ ] **Step 5: Validate and commit**

Run:

```bash
cd /Users/mat/Desktop/media-library-manager/backend
go test ./internal/campaigns ./internal/recommendations -count=1
```

Commit:

```bash
git add backend/internal/campaigns backend/internal/recommendations/engine.go
git commit -m "feat: add stewardship campaign engine"
```

## Task 2: Campaign Persistence

**Files:**
- Modify: `backend/internal/database/store.go`
- Create: `backend/internal/database/campaigns_test.go`

- [ ] **Step 1: Write failing persistence tests**

Add tests for:

- campaign upsert/list/get/delete round trip
- campaign run record round trip
- campaign recommendation replacement only replaces open recommendations from the same campaign source

Run:

```bash
cd /Users/mat/Desktop/media-library-manager/backend
go test ./internal/database -run TestCampaign -count=1
```

Expected: fail because store methods do not exist.

- [ ] **Step 2: Add schema and store methods**

Add tables:

```sql
CREATE TABLE IF NOT EXISTS stewardship_campaigns (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  enabled INTEGER NOT NULL DEFAULT 1,
  target_kinds TEXT NOT NULL DEFAULT '[]',
  target_library_names TEXT NOT NULL DEFAULT '[]',
  require_all_rules INTEGER NOT NULL DEFAULT 1,
  minimum_confidence REAL NOT NULL DEFAULT 0,
  minimum_storage_bytes INTEGER NOT NULL DEFAULT 0,
  rules TEXT NOT NULL DEFAULT '[]',
  last_run_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS stewardship_campaign_runs (
  id TEXT PRIMARY KEY,
  campaign_id TEXT NOT NULL,
  status TEXT NOT NULL,
  matched INTEGER NOT NULL DEFAULT 0,
  suppressed INTEGER NOT NULL DEFAULT 0,
  estimated_savings_bytes INTEGER NOT NULL DEFAULT 0,
  verified_savings_bytes INTEGER NOT NULL DEFAULT 0,
  error TEXT NOT NULL DEFAULT '',
  started_at TEXT NOT NULL,
  completed_at TEXT
);
```

Implement store methods:

- `UpsertCampaign(campaigns.Campaign) (campaigns.Campaign, error)`
- `ListCampaigns() ([]campaigns.Campaign, error)`
- `GetCampaign(id string) (campaigns.Campaign, error)`
- `DeleteCampaign(id string) error`
- `RecordCampaignRun(campaigns.Run) error`
- `ListCampaignRuns(campaignID string) ([]campaigns.Run, error)`
- `ReplaceCampaignRecommendations(campaignID string, recs []recommendations.Recommendation) error`

- [ ] **Step 3: Validate and commit**

Run:

```bash
cd /Users/mat/Desktop/media-library-manager/backend
go test ./internal/database -run TestCampaign -count=1
```

Commit:

```bash
git add backend/internal/database/store.go backend/internal/database/campaigns_test.go
git commit -m "feat: persist stewardship campaigns"
```

## Task 3: Campaign API

**Files:**
- Modify: `backend/internal/api/server.go`
- Create: `backend/internal/api/campaigns_test.go`

- [ ] **Step 1: Write failing API tests**

Add tests proving:

- `POST /api/v1/campaigns` creates a campaign
- `GET /api/v1/campaigns` lists it
- `POST /api/v1/campaigns/{id}/simulate` returns matched/suppressed counts
- `POST /api/v1/campaigns/{id}/run` creates non-destructive campaign recommendations
- `DELETE /api/v1/campaigns/{id}` deletes only the campaign definition

Run:

```bash
cd /Users/mat/Desktop/media-library-manager/backend
go test ./internal/api -run TestCampaign -count=1
```

Expected: fail because routes do not exist.

- [ ] **Step 2: Implement API routes**

Add handlers under `/api/v1/campaigns`. Use `store.ListActivityRecommendationMedia()` as the candidate source. Convert activity rows to campaign candidates through `campaigns.CandidatesFromActivity`.

- [ ] **Step 3: Validate and commit**

Run:

```bash
cd /Users/mat/Desktop/media-library-manager/backend
go test ./internal/api ./internal/database ./internal/campaigns -count=1
```

Commit:

```bash
git add backend/internal/api/server.go backend/internal/api/campaigns_test.go
git commit -m "feat: expose stewardship campaign api"
```

## Task 4: Frontend API And Types

**Files:**
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/lib/api.ts`
- Modify: `frontend/src/lib/api.test.ts`

- [ ] **Step 1: Write failing frontend API tests**

Add tests for campaign list/create/update/delete/simulate/run endpoints.

Run:

```bash
cd /Users/mat/Desktop/media-library-manager
npm --prefix frontend run test -- --run src/lib/api.test.ts
```

Expected: fail because API helpers do not exist.

- [ ] **Step 2: Add types and API helpers**

Add TypeScript types for `Campaign`, `CampaignRule`, `CampaignResult`, `CampaignRun`, and helper methods:

- `campaigns()`
- `createCampaign(input)`
- `updateCampaign(id, input)`
- `deleteCampaign(id)`
- `simulateCampaign(id)`
- `runCampaign(id)`
- `campaignRuns(id)`

- [ ] **Step 3: Validate and commit**

Run:

```bash
npm --prefix frontend run test -- --run src/lib/api.test.ts
npm --prefix frontend run build
```

Commit:

```bash
git add frontend/src/types.ts frontend/src/lib/api.ts frontend/src/lib/api.test.ts
git commit -m "feat: add campaign frontend api client"
```

## Task 5: Campaigns UI

**Files:**
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/styles/app.css`

- [ ] **Step 1: Add Campaigns navigation and state**

Add `campaigns` to the `View` union and navigation. Load campaigns and keep latest simulation result in state.

- [ ] **Step 2: Add campaign list and editor**

Create a compact Campaigns view with:

- list of campaigns
- create/update form
- enabled toggle
- target kind checkboxes
- minimum confidence and storage fields
- rule rows for common fields/operators

- [ ] **Step 3: Add simulation and run panels**

Add buttons for `Simulate` and `Run`. Show matched count, suppressed count, estimated savings, verified savings, confidence range, and first matched/suppressed rows.

- [ ] **Step 4: Validate and commit**

Run:

```bash
npm --prefix frontend run test -- --run
npm --prefix frontend run build
```

Commit:

```bash
git add frontend/src/App.tsx frontend/src/styles/app.css
git commit -m "feat: add stewardship campaigns ui"
```

## Task 6: Review Queue Integration And Docs

**Files:**
- Modify: `frontend/src/App.tsx`
- Modify: `README.md`
- Modify: `docs/api/rest.md`
- Modify: `docs/recommendation-proof.md`

- [ ] **Step 1: Show campaign evidence in review cards**

In the proof summary, show campaign name/run when `rec.evidence.campaignName` exists.

- [ ] **Step 2: Document campaigns**

Document:

- what campaigns do
- simulation-first safety
- API routes
- no media deletion

- [ ] **Step 3: Validate and commit**

Run:

```bash
make ci
```

Commit:

```bash
git add frontend/src/App.tsx README.md docs/api/rest.md docs/recommendation-proof.md
git commit -m "docs: document stewardship campaigns"
```

## Release Gate

Run:

```bash
cd /Users/mat/Desktop/media-library-manager
make ci
docker compose config --quiet
docker compose --profile ai config --quiet
docker compose --profile ai up -d --build
curl -fsS http://localhost:8080/api/v1/health
scripts/verify-no-delete.sh
```

Expected:

- backend tests pass
- frontend tests pass
- production frontend build passes
- Go vet passes
- Docker image builds
- no-delete invariant passes
- app health endpoint returns OK

## Commit Strategy

Commit after every task above. If a task grows, split it further into commits such as:

- `test: cover campaign rule operators`
- `feat: evaluate campaign rule operators`
- `feat: persist stewardship campaigns`
- `feat: expose campaign simulation api`
- `feat: add campaigns ui shell`
- `docs: document stewardship campaigns`
