package jj

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubstitutePlaceholders_SinglePlaceholder(t *testing.T) {
	out := SubstitutePlaceholders(
		map[string]string{"DFT_WIDTH": "$preview_width"},
		map[string]string{PreviewWidthPlaceholder: "80"},
	)
	assert.Equal(t, []string{"DFT_WIDTH=80"}, out)
}

func TestSubstitutePlaceholders_MultiplePlaceholdersInOneValue(t *testing.T) {
	out := SubstitutePlaceholders(
		map[string]string{"COMBO": "$change_id-$commit_id"},
		map[string]string{
			ChangeIdPlaceholder: "abc",
			CommitIdPlaceholder: "deadbeef",
		},
	)
	assert.Equal(t, []string{"COMBO=abc-deadbeef"}, out)
}

func TestSubstitutePlaceholders_UnknownPlaceholderPassesThrough(t *testing.T) {
	out := SubstitutePlaceholders(
		map[string]string{"X": "$foo"},
		map[string]string{PreviewWidthPlaceholder: "80"},
	)
	assert.Equal(t, []string{"X=$foo"}, out)
}

func TestSubstitutePlaceholders_EmptyValue(t *testing.T) {
	out := SubstitutePlaceholders(
		map[string]string{"X": ""},
		map[string]string{PreviewWidthPlaceholder: "80"},
	)
	assert.Equal(t, []string{"X="}, out)
}

func TestSubstitutePlaceholders_NilValues(t *testing.T) {
	out := SubstitutePlaceholders(nil, map[string]string{PreviewWidthPlaceholder: "80"})
	assert.Empty(t, out)
}

func TestSubstitutePlaceholders_EmptyValues(t *testing.T) {
	out := SubstitutePlaceholders(map[string]string{}, map[string]string{PreviewWidthPlaceholder: "80"})
	assert.Empty(t, out)
}

func TestSubstitutePlaceholders_DoesNotMutateReplacements(t *testing.T) {
	replacements := map[string]string{
		FilePlaceholder:         "path/with space.txt",
		PreviewWidthPlaceholder: "80",
	}
	snapshot := map[string]string{}
	for k, v := range replacements {
		snapshot[k] = v
	}
	_ = SubstitutePlaceholders(
		map[string]string{"X": "$file", "Y": "$preview_width"},
		replacements,
	)
	assert.Equal(t, snapshot, replacements)
}

func TestSubstitutePlaceholders_ValueWithoutPlaceholder(t *testing.T) {
	out := SubstitutePlaceholders(
		map[string]string{"X": "literal"},
		map[string]string{PreviewWidthPlaceholder: "80"},
	)
	assert.Equal(t, []string{"X=literal"}, out)
}

func TestSubstitutePlaceholders_ValueEqualsPlaceholder(t *testing.T) {
	out := SubstitutePlaceholders(
		map[string]string{"V": "$preview_width"},
		map[string]string{PreviewWidthPlaceholder: "80"},
	)
	assert.Equal(t, []string{"V=80"}, out)
}

func TestSubstitutePlaceholders_DeterministicOrdering(t *testing.T) {
	out := SubstitutePlaceholders(
		map[string]string{"B": "2", "A": "1", "C": "3"},
		nil,
	)
	assert.Equal(t, []string{"A=1", "B=2", "C=3"}, out)
}

func TestBookmarkPatternCommandsUseExactStringPatterns(t *testing.T) {
	name := `1.3.63-+-json-length-"fix"\branch`
	remote := `origin+backup`

	assert.Equal(t, CommandArgs{"bookmark", "move", `exact:"1.3.63-+-json-length-\"fix\"\\branch"`, "--to", "abc123"}, BookmarkMove("abc123", name))
	assert.Equal(t, CommandArgs{"bookmark", "delete", `exact:"1.3.63-+-json-length-\"fix\"\\branch"`}, BookmarkDelete(name))
	assert.Equal(t, CommandArgs{"bookmark", "forget", `exact:"1.3.63-+-json-length-\"fix\"\\branch"`}, BookmarkForget(name))
	assert.Equal(t, CommandArgs{"bookmark", "track", `exact:"1.3.63-+-json-length-\"fix\"\\branch"`, "--remote", `exact:"origin+backup"`}, BookmarkTrack(name, remote))
	assert.Equal(t, CommandArgs{"bookmark", "untrack", `exact:"1.3.63-+-json-length-\"fix\"\\branch"`, "--remote", `exact:"origin+backup"`}, BookmarkUntrack(name, remote))
}
