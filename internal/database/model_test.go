package database

import (
	"database/sql"
	"testing"
)

func TestSetGetRepositories_RoundTrip(t *testing.T) {
	badge := &Badge{}
	repos := []Repository{
		{Name: "Frontend", URL: "https://github.com/org/frontend"},
		{Name: "Backend", URL: "https://github.com/org/backend"},
	}

	if err := badge.SetRepositories(repos); err != nil {
		t.Fatalf("SetRepositories failed: %v", err)
	}

	got := badge.GetRepositories()
	if len(got) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(got))
	}
	if got[0].Name != "Frontend" || got[0].URL != "https://github.com/org/frontend" {
		t.Errorf("repo[0] mismatch: %+v", got[0])
	}
	if got[1].Name != "Backend" || got[1].URL != "https://github.com/org/backend" {
		t.Errorf("repo[1] mismatch: %+v", got[1])
	}
}

func TestGetRepositories_BackwardCompat_PlainURL(t *testing.T) {
	badge := &Badge{
		RepositoryLink: sql.NullString{
			String: "https://github.com/org/repo",
			Valid:  true,
		},
	}

	got := badge.GetRepositories()
	if len(got) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(got))
	}
	if got[0].Name != "https://github.com/org/repo" {
		t.Errorf("expected Name to equal URL, got %q", got[0].Name)
	}
	if got[0].URL != "https://github.com/org/repo" {
		t.Errorf("expected URL %q, got %q", "https://github.com/org/repo", got[0].URL)
	}
}

func TestGetRepositories_Empty(t *testing.T) {
	tests := []struct {
		name  string
		badge Badge
	}{
		{"invalid NullString", Badge{RepositoryLink: sql.NullString{Valid: false}}},
		{"empty string", Badge{RepositoryLink: sql.NullString{String: "", Valid: true}}},
		{"whitespace only", Badge{RepositoryLink: sql.NullString{String: "   ", Valid: true}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.badge.GetRepositories()
			if got != nil {
				t.Errorf("expected nil, got %+v", got)
			}
		})
	}
}

func TestSetRepositories_Empty(t *testing.T) {
	badge := &Badge{
		RepositoryLink: sql.NullString{String: "old", Valid: true},
	}

	if err := badge.SetRepositories(nil); err != nil {
		t.Fatalf("SetRepositories(nil) failed: %v", err)
	}
	if badge.RepositoryLink.Valid {
		t.Error("expected RepositoryLink to be invalid after setting nil repos")
	}
}

func TestSetRepositories_EmptySlice(t *testing.T) {
	badge := &Badge{}
	if err := badge.SetRepositories([]Repository{}); err != nil {
		t.Fatalf("SetRepositories([]) failed: %v", err)
	}
	if badge.RepositoryLink.Valid {
		t.Error("expected RepositoryLink to be invalid after setting empty slice")
	}
}
