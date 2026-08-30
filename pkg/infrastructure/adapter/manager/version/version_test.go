package version

import (
	"testing"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
)

const (
	fixtureVersion100  = "1.0.0"
	fixtureVersion1053 = "10.5.3"
)

func TestDetermineUpdateType(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    manager.UpdateType
	}{
		{
			name:    "major version change",
			current: fixtureVersion100,
			latest:  "2.0.0",
			want:    manager.UpdateMajor,
		},
		{
			name:    "minor version change",
			current: fixtureVersion100,
			latest:  "1.1.0",
			want:    manager.UpdateMinor,
		},
		{
			name:    "patch version change",
			current: fixtureVersion100,
			latest:  "1.0.1",
			want:    manager.UpdatePatch,
		},
		{
			name:    "with v prefix",
			current: "v1.0.0",
			latest:  "v2.0.0",
			want:    manager.UpdateMajor,
		},
		{
			name:    "mixed v prefix",
			current: "v1.0.0",
			latest:  "2.0.0",
			want:    manager.UpdateMajor,
		},
		{
			name:    "same version",
			current: fixtureVersion100,
			latest:  fixtureVersion100,
			want:    manager.UpdatePatch,
		},
		{
			name:    "short version format major",
			current: "1.0",
			latest:  "2.0",
			want:    manager.UpdateMajor,
		},
		{
			name:    "short version format minor",
			current: "1.0",
			latest:  "1.1",
			want:    manager.UpdateMinor,
		},
		{
			name:    "single digit version",
			current: "1",
			latest:  "2",
			want:    manager.UpdateMajor,
		},
		{
			name:    "empty current",
			current: "",
			latest:  fixtureVersion100,
			want:    manager.UpdateMajor,
		},
		{
			name:    "empty latest",
			current: fixtureVersion100,
			latest:  "",
			want:    manager.UpdateMajor,
		},
		{
			name:    "both empty",
			current: "",
			latest:  "",
			want:    manager.UpdatePatch,
		},
		{
			name:    "complex version major",
			current: fixtureVersion1053,
			latest:  "11.0.0",
			want:    manager.UpdateMajor,
		},
		{
			name:    "complex version minor",
			current: fixtureVersion1053,
			latest:  "10.6.0",
			want:    manager.UpdateMinor,
		},
		{
			name:    "complex version patch",
			current: fixtureVersion1053,
			latest:  "10.5.4",
			want:    manager.UpdatePatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetermineUpdateType(tt.current, tt.latest)
			if got != tt.want {
				t.Errorf("DetermineUpdateType(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.want)
			}
		})
	}
}
