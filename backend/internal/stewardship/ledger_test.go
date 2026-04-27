package stewardship

import "testing"

func TestBuildStorageLedgerSeparatesTruthBuckets(t *testing.T) {
	ledger := BuildStorageLedger(LedgerInput{
		Recommendations: []LedgerRecommendation{
			{ID: "local", State: "new", EstimatedBytes: 100, VerifiedBytes: 100, Verification: "local_verified"},
			{ID: "mapped", State: "new", EstimatedBytes: 80, Verification: "path_mapped"},
			{ID: "server", State: "new", EstimatedBytes: 60, Verification: "server_reported"},
			{ID: "blocked", State: "new", EstimatedBytes: 40, Verification: "unmapped"},
			{ID: "protected", State: "protected", EstimatedBytes: 30, VerifiedBytes: 30, Verification: "local_verified"},
			{ID: "accepted", State: "accepted_for_manual_action", EstimatedBytes: 20, VerifiedBytes: 20, Verification: "local_verified"},
		},
		RequestSignals: []RequestSignal{{ExternalRequestID: "r1", EstimatedBytes: 25}},
	})

	if ledger.LocallyVerifiedBytes != 100 {
		t.Fatalf("LocallyVerifiedBytes = %d, want 100", ledger.LocallyVerifiedBytes)
	}
	if ledger.MappedEstimateBytes != 80 || ledger.ServerReportedBytes != 60 || ledger.BlockedUnmappedBytes != 40 {
		t.Fatalf("unexpected estimate buckets: %#v", ledger)
	}
	if ledger.ProtectedBytes != 30 || ledger.AcceptedManualBytes != 20 || ledger.RequestedMediaBytes != 25 {
		t.Fatalf("unexpected state buckets: %#v", ledger)
	}
}

func TestBuildWhatIfSimulationAggregatesCampaignResults(t *testing.T) {
	result := BuildWhatIfSimulation([]CampaignSimulation{
		{
			CampaignID:          "cmp_1",
			Matched:             2,
			Suppressed:          1,
			EstimatedBytes:      300,
			VerifiedBytes:       120,
			BlockedUnmapped:     2,
			ProtectionConflicts: 1,
			RequestConflicts:    1,
		},
	})

	if result.Campaigns != 1 || result.Matched != 2 || result.Suppressed != 1 {
		t.Fatalf("unexpected counts: %#v", result)
	}
	if result.EstimatedBytes != 300 || result.VerifiedBytes != 120 || result.BlockedUnmapped != 2 {
		t.Fatalf("unexpected bytes/blockers: %#v", result)
	}
	if result.ProtectionConflicts != 1 || result.RequestConflicts != 1 {
		t.Fatalf("unexpected conflicts: %#v", result)
	}
}
