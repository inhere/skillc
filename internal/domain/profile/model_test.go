package profile

import (
	"testing"

	"github.com/gookit/goutil/x/assert"
)

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{name: "go-dev"},
		{name: "flutter_dev"},
		{name: "security.review"},
		{name: "", wantErr: true},
		{name: "has space", wantErr: true},
		{name: "../bad", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.name)
			if tt.wantErr {
				assert.NotNil(t, err)
				return
			}
			assert.NoErr(t, err)
		})
	}
}

func TestNormalizeTargetsSortsAndDeduplicates(t *testing.T) {
	targets := []Target{
		{Source: "gstack", Skill: "review"},
		{Source: "gstack", Skill: "go-pro"},
		{Source: "gstack", Skill: "review"},
		{Skill: "local-only"},
	}

	got, err := NormalizeTargets(targets)

	assert.NoErr(t, err)
	assert.Len(t, got, 3)
	assert.Eq(t, "local-only", got[0].Skill)
	assert.Eq(t, "go-pro", got[1].Skill)
	assert.Eq(t, "review", got[2].Skill)
}

func TestNormalizeTargetsRejectsEmptySkill(t *testing.T) {
	_, err := NormalizeTargets([]Target{{Source: "gstack"}})

	assert.NotNil(t, err)
}
