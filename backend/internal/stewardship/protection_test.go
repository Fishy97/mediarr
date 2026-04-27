package stewardship

import "testing"

func TestProtectionRequestApproveAndDeclineTransitions(t *testing.T) {
	request := ProtectionRequest{Title: "The Sopranos", RequestedBy: "Alex"}.WithDefaults()
	approved, err := request.Approve("admin", "family favorite")
	if err != nil {
		t.Fatalf("Approve returned error: %v", err)
	}
	if approved.Status != "approved" || approved.DecisionBy != "admin" || approved.DecisionNote != "family favorite" {
		t.Fatalf("unexpected approved request: %#v", approved)
	}
	if _, err := approved.Decline("admin", "changed mind"); err == nil {
		t.Fatal("approved request should not be declined")
	}

	request = ProtectionRequest{Title: "Old Movie", RequestedBy: "Sam"}.WithDefaults()
	declined, err := request.Decline("admin", "duplicate request")
	if err != nil {
		t.Fatalf("Decline returned error: %v", err)
	}
	if declined.Status != "declined" || declined.DecisionBy != "admin" {
		t.Fatalf("unexpected declined request: %#v", declined)
	}
}
