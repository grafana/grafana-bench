package git

import (
	"testing"
)

func TestNanoGitAuthSetup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping auth setup tests in short mode")
	}
	
	tests := []struct {
		name     string
		repo     string
		token    string
		wantErr  bool
	}{
		{
			name:    "public repository - no auth",
			repo:    "https://github.com/octocat/Hello-World",
			token:   "",
			wantErr: false,
		},
		{
			name:    "private repository - with token",
			repo:    "https://github.com/private/repo",
			token:   "ghp_test_token_123456789",
			wantErr: false,
		},
		{
			name:    "invalid repository URL",
			repo:    "not-a-valid-url",
			token:   "",
			wantErr: true,
		},
		{
			name:    "private repository with token auth",
			repo:    "https://gitlab.com/private/project",
			token:   "glpat_test_token_123456789",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nanoRepo, err := NewNanoGitSource(tt.repo, tt.token)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewNanoGitSource() error = nil, wantErr %v", tt.wantErr)
				}
				return
			}
			
			if err != nil {
				t.Errorf("NewNanoGitSource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			// Verify the client was created
			if nanoRepo == nil {
				t.Error("NewNanoGitSource() returned nil repo")
			}
			
			// Verify the token was stored
			if nanoRepo.RepoToken != tt.token {
				t.Errorf("NewNanoGitSource() stored token = %v, want %v", nanoRepo.RepoToken, tt.token)
			}
			
			// Verify the repository URL was stored
			if nanoRepo.Repo != tt.repo {
				t.Errorf("NewNanoGitSource() stored repo = %v, want %v", nanoRepo.Repo, tt.repo)
			}
		})
	}
}