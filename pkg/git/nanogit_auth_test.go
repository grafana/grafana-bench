package git

import (
	"testing"
)

func TestGitAuthSetup(t *testing.T) {
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
			gitRepo, err := NewGitSource(tt.repo, tt.token)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewGitSource() error = nil, wantErr %v", tt.wantErr)
				}
				return
			}
			
			if err != nil {
				t.Errorf("NewGitSource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			// Verify the client was created
			if gitRepo == nil {
				t.Error("NewGitSource() returned nil repo")
			}
			
			// Verify the token was stored
			if gitRepo.RepoToken != tt.token {
				t.Errorf("NewGitSource() stored token = %v, want %v", gitRepo.RepoToken, tt.token)
			}
			
			// Verify the repository URL was stored
			if gitRepo.Repo != tt.repo {
				t.Errorf("NewGitSource() stored repo = %v, want %v", gitRepo.Repo, tt.repo)
			}
		})
	}
}