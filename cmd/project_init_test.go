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
			url:       "https://github.com/users/maxbeizer/projects/24",
			wantOwner: "maxbeizer",
			wantBoard: 24,
		},
		{
			name:      "org project URL",
			url:       "https://github.com/orgs/github/projects/24127",
			wantOwner: "github",
			wantBoard: 24127,
		},
		{
			name:      "trailing slash",
			url:       "https://github.com/users/maxbeizer/projects/24/",
			wantOwner: "maxbeizer",
			wantBoard: 24,
		},
		{
			name:      "with view suffix",
			url:       "https://github.com/users/maxbeizer/projects/24/views/1",
			wantOwner: "maxbeizer",
			wantBoard: 24,
		},
		{
			name:    "missing projects segment",
			url:     "https://github.com/maxbeizer/gh-helm",
			wantErr: true,
		},
		{
			name:    "non-numeric board",
			url:     "https://github.com/users/maxbeizer/projects/abc",
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
