package review

import (
	"testing"
)

func TestInjectCredentials(t *testing.T) {
	tests := []struct {
		name     string
		repoURL  string
		user     string
		password string
		want     string
	}{
		{
			name:     "https with credentials",
			repoURL:  "https://gitlab.com/org/repo.git",
			user:     "alice",
			password: "secret-token",
			want:     "https://alice:secret-token@gitlab.com/org/repo.git",
		},
		{
			name:     "http with credentials",
			repoURL:  "http://git.example.com/repo.git",
			user:     "bob",
			password: "pass123",
			want:     "http://bob:pass123@git.example.com/repo.git",
		},
		{
			name:     "empty credentials returns original",
			repoURL:  "https://gitlab.com/org/repo.git",
			user:     "",
			password: "",
			want:     "https://gitlab.com/org/repo.git",
		},
		{
			name:     "ssh url unchanged",
			repoURL:  "git@gitlab.com:org/repo.git",
			user:     "alice",
			password: "token",
			want:     "git@gitlab.com:org/repo.git",
		},
		{
			name:     "url with existing userinfo replaced",
			repoURL:  "https://old:oldpass@gitlab.com/org/repo.git",
			user:     "new",
			password: "newpass",
			want:     "https://new:newpass@gitlab.com/org/repo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := injectCredentials(tt.repoURL, tt.user, tt.password)
			if got != tt.want {
				t.Errorf("injectCredentials(%q, %q, %q) = %q, want %q", tt.repoURL, tt.user, tt.password, got, tt.want)
			}
		})
	}
}
