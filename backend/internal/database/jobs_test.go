package database

import "testing"

func TestJobsPersistProgressAndEvents(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	job, err := store.CreateJob(JobInput{
		Kind:     "filesystem_scan",
		TargetID: "all",
		Phase:    "queued",
		Message:  "Scan queued",
		Total:    3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if job.ID == "" || job.Status != "queued" || job.StartedAt.IsZero() || job.UpdatedAt.IsZero() {
		t.Fatalf("created job = %#v", job)
	}

	updated, err := store.UpdateJob(job.ID, JobUpdate{
		Status:       "running",
		Phase:        "processing",
		Message:      "Processing media",
		CurrentLabel: "Arrival (2016).mkv",
		Processed:    intPointer(1),
		Total:        intPointer(3),
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != "running" || updated.Processed != 1 || updated.Total != 3 || updated.CurrentLabel != "Arrival (2016).mkv" {
		t.Fatalf("updated job = %#v", updated)
	}

	if _, err := store.AddJobEvent(JobEventInput{
		JobID:        job.ID,
		Level:        "info",
		Phase:        "processing",
		Message:      "Processed Arrival (2016).mkv",
		CurrentLabel: "Arrival (2016).mkv",
		Processed:    1,
		Total:        3,
	}); err != nil {
		t.Fatal(err)
	}

	events, err := store.ListJobEvents(job.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Message != "Processed Arrival (2016).mkv" {
		t.Fatalf("events = %#v", events)
	}

	completed, err := store.UpdateJob(job.ID, JobUpdate{
		Status:    "completed",
		Phase:     "complete",
		Message:   "Scan completed",
		Completed: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != "completed" || completed.CompletedAt.IsZero() {
		t.Fatalf("completed job = %#v", completed)
	}

	found, err := store.GetJob(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found.ID != job.ID || found.Status != "completed" {
		t.Fatalf("found job = %#v", found)
	}

	latest, err := store.LatestJob("filesystem_scan", "all")
	if err != nil {
		t.Fatal(err)
	}
	if latest.ID != job.ID {
		t.Fatalf("latest = %#v", latest)
	}

	jobs, err := store.ListJobs(JobFilter{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 || jobs[0].ID != job.ID {
		t.Fatalf("jobs = %#v", jobs)
	}
}

func intPointer(value int) *int {
	return &value
}
