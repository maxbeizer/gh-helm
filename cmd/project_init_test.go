package cmd

import "testing"

func TestParseProjectURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantOwner string
		wantBoard int
		wantErr   bool
	}{
		{
			name:      "user project URL",
			url:       "https://github.com/users/octocat/projects/1",
			wantOwner: "octocat",
			wantBoard: 1,
		},
		{
			name:      "org project URL",
			url:       "https://github.com/orgs/my-org/projects/42",
			wantOwner: "my-org",
			wantBoard: 42,
		},
		{
			name:      "trailing slash",
			url:       "https://github.com/users/octocat/projects/1/",
			wantOwner: "octocat",
			wantBoard: 1,
		},
		{
			name:      "with view suffix",
			url:       "https://github.com/users/octocat/projects/1/views/1",
			wantOwner: "octocat",
			wantBoard: 1,
		},
		{
			name:    "missing projects segment",
			url:     "https://github.com/octocat/hello-world",
			wantErr: true,
		},
		{
			name:    "non-numeric board",
			url:     "https://github.com/users/octocat/projects/abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, board, err := parseProjectURL(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if owner != tt.wantOwner {
				t.Errorf("owner = %q, want %q", owner, tt.wantOwner)
			}
			if board != tt.wantBoard {
				t.Errorf("board = %d, want %d", board, tt.wantBoard)
			}
		})
	}
}
