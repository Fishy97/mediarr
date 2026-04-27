package stewardship

import "strings"

func BuildStorageLedger(input LedgerInput) StorageLedger {
	var ledger StorageLedger
	for _, rec := range input.Recommendations {
		estimated := maxInt64(rec.EstimatedBytes, 0)
		verified := maxInt64(rec.VerifiedBytes, 0)
		state := strings.TrimSpace(rec.State)
		switch state {
		case "protected":
			ledger.ProtectedBytes += estimated
			continue
		case "accepted_for_manual_action":
			ledger.AcceptedManualBytes += estimated
			continue
		}
		ledger.TotalEstimatedBytes += estimated
		ledger.TotalVerifiedBytes += verified
		switch strings.TrimSpace(rec.Verification) {
		case "local_verified":
			if verified > 0 {
				ledger.LocallyVerifiedBytes += verified
			} else {
				ledger.LocallyVerifiedBytes += estimated
			}
		case "path_mapped":
			ledger.MappedEstimateBytes += estimated
		case "server_reported":
			ledger.ServerReportedBytes += estimated
		default:
			ledger.BlockedUnmappedBytes += estimated
		}
	}
	for _, signal := range input.RequestSignals {
		ledger.RequestedMediaBytes += maxInt64(signal.EstimatedBytes, 0)
	}
	return ledger
}

func BuildWhatIfSimulation(simulations []CampaignSimulation) WhatIfSimulation {
	var result WhatIfSimulation
	result.Campaigns = len(simulations)
	for _, simulation := range simulations {
		result.Matched += simulation.Matched
		result.Suppressed += simulation.Suppressed
		result.EstimatedBytes += maxInt64(simulation.EstimatedBytes, 0)
		result.VerifiedBytes += maxInt64(simulation.VerifiedBytes, 0)
		result.BlockedUnmapped += simulation.BlockedUnmapped
		result.ProtectionConflicts += simulation.ProtectionConflicts
		result.RequestConflicts += simulation.RequestConflicts
	}
	return result
}

func maxInt64(value int64, floor int64) int64 {
	if value < floor {
		return floor
	}
	return value
}
