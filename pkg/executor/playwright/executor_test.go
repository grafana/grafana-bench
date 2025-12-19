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
