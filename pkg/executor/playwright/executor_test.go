package playwright

import (
	"testing"
)

func TestAppendReporterToCommand(t *testing.T) {
	tests := []struct {
		name       string
		executeCmd string
		suitePath  string
		want       string
	}{
		{
			name:       "npm run command",
			executeCmd: "npm run test",
			suitePath:  "tests",
			want:       "npm run test -- --reporter=json tests",
		},
		{
			name:       "npm run in longer command",
			executeCmd: "NODE_ENV=ci npm run test:e2e",
			suitePath:  "e2e",
			want:       "NODE_ENV=ci npm run test:e2e -- --reporter=json e2e",
		},
		{
			name:       "yarn command",
			executeCmd: "yarn test",
			suitePath:  "tests",
			want:       "yarn test --reporter=json tests",
		},
		{
			name:       "pnpm command",
			executeCmd: "pnpm test",
			suitePath:  "tests",
			want:       "pnpm test --reporter=json tests",
		},
		{
			name:       "other command",
			executeCmd: "playwright test",
			suitePath:  "tests",
			want:       "playwright test --reporter=json tests",
		},
		{
			name:       "npm run with its own separator keeps a single one",
			executeCmd: "npm run e2e -- --grep-invert @quarantine",
			suitePath:  "/tests",
			want:       "npm run e2e -- --grep-invert @quarantine --reporter=json /tests",
		},
		{
			name:       "npm run ending in a separator",
			executeCmd: "npm run e2e --",
			suitePath:  "tests",
			want:       "npm run e2e -- --reporter=json tests",
		},
		{
			name:       "npm run with a double dash inside a flag value",
			executeCmd: "npm run e2e:cloud --if-present",
			suitePath:  "tests",
			want:       "npm run e2e:cloud --if-present -- --reporter=json tests",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appendReporterToCommand(tt.executeCmd, tt.suitePath)
			if got != tt.want {
				t.Fatalf("appendReporterToCommand(%q, %q) = %q, want %q",
					tt.executeCmd, tt.suitePath, got, tt.want)
			}
		})
	}
}

func TestWorkDir(t *testing.T) {
	tests := []struct {
		name      string
		baseDir   string
		suitePath string
		want      string
	}{
		{
			name:      "relative suite path joins the base directory",
			baseDir:   "/work",
			suitePath: "tests",
			want:      "/work/tests",
		},
		{
			name:      "current directory for both",
			baseDir:   ".",
			suitePath: ".",
			want:      ".",
		},
		{
			name:      "absolute suite path stands alone",
			baseDir:   ".",
			suitePath: "/tests",
			want:      "/tests",
		},
		{
			name:      "absolute suite path ignores an absolute base directory",
			baseDir:   "/work",
			suitePath: "/tests",
			want:      "/tests",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := workDir(tt.baseDir, tt.suitePath)
			if got != tt.want {
				t.Fatalf("workDir(%q, %q) = %q, want %q", tt.baseDir, tt.suitePath, got, tt.want)
			}
		})
	}
}
