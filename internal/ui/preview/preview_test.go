package preview

import (
	"fmt"
	"testing"

	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"

	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModel_Init(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := New(ctx)

	test.SimulateModel(model, model.Init())
}

func TestModel_View(t *testing.T) {
	tests := []struct {
		name     string
		scrollBy layout.Position
		atBottom bool
		width    int
		height   int
		content  string
		expected string
	}{
		{
			name:     "clips",
			scrollBy: layout.Position{},
			width:    5,
			height:   2,
			content: test.Stripped(`
			+++++..
			+abcde.
			+++++..
			`),
			expected: test.Stripped(`
			+++++
			+abcd
			`),
		},
		{
			name:     "clips when at bottom",
			scrollBy: layout.Position{},
			atBottom: true,
			width:    5,
			height:   3,
			content: test.Stripped(`
			+++++..
			+abcde.
			+++++..
			`),
			expected: test.Stripped(`
			+++++
			+abcd
			+++++
			`),
		},
		{
			name:     "Scroll by down and right",
			scrollBy: layout.Position{X: 1, Y: 1},
			width:    5,
			height:   2,
			content: test.Stripped(`
			.......
			.abcde.
			.......
			`),
			expected: test.Stripped(`
			abcde
			.....
			`),
		},
		{
			name:     "Scroll down when at bottom",
			scrollBy: layout.Position{X: 0, Y: 1},
			atBottom: true,
			width:    5,
			height:   3,
			content: test.Stripped(`
			.......
			.abcde.
			.......
			`),
			expected: test.Stripped(`
			.abcd
			.....
			`),
		},
		{
			name:     "Scroll 2 right when at bottom",
			scrollBy: layout.Position{X: 2, Y: 0},
			atBottom: true,
			width:    5,
			height:   3,
			content: test.Stripped(`
			.......
			.abcde.
			.......
			`),
			expected: test.Stripped(`
			.....
			bcde.
			.....
			`),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := test.NewTestContext(test.NewTestCommandRunner(t))

			model := New(ctx)

			model.previewAtBottom = tc.atBottom
			model.SetContent(tc.content)
			if tc.scrollBy.X > 0 {
				model.ScrollHorizontal(tc.scrollBy.X)
			}
			if tc.scrollBy.Y > 0 {
				model.Scroll(tc.scrollBy.Y)
			}
			v := test.Stripped(test.RenderImmediate(model, tc.width, tc.height))

			assert.Equal(t, tc.expected, v)
		})
	}
}

func TestUpdate_PreviewShowResetsScroll(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)

	model.SetContent("first line\nsecond line\nthird line")
	model.Scroll(2)
	model.ScrollHorizontal(2)

	cmd := model.Update(intents.PreviewShow{Content: "updated"})
	require.Nil(t, cmd)

	assert.Equal(t, 0, model.view.YOffset())
	assert.Equal(t, 0, model.view.XOffset())
	assert.Equal(t, "updated", model.content)
}

func TestUpdate_PreviewShowDoesNotBreakSelectionRefresh(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	ctx.CurrentRevset = "all()"
	model := New(ctx)

	selected := common.SelectedRevision{ChangeId: "change", CommitId: "commit"}
	args := jj.TemplatedArgs(config.Current.Preview.RevisionCommand, map[string]string{
		jj.RevsetPlaceholder:       ctx.CurrentRevset,
		jj.ChangeIdPlaceholder:     selected.ChangeId,
		jj.CommitIdPlaceholder:     selected.CommitId,
		jj.PreviewWidthPlaceholder: "0",
	})
	commandRunner.Expect(args).SetOutput([]byte("auto preview"))

	model.Update(intents.PreviewShow{Content: "manual"})
	cmd := model.Update(common.SelectionChangedMsg{Item: selected})
	require.NotNil(t, cmd)

	test.SimulateModel(model, cmd)

	assert.Equal(t, "auto preview", model.content)
}

func TestSetContent_ExpandsTabsUsingTabStops(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)

	model.SetContent("+\tfoo")

	rendered := test.RenderImmediate(model, 12, 1)
	assert.Equal(t, "+   foo", rendered)
}

func TestSetContent_ResetsTabStopsAfterNewlines(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)

	model.SetContent("a\tb\nab\tc")

	rendered := test.RenderImmediate(model, 12, 2)
	assert.Equal(t, "a   b\nab  c", rendered)
}

func baseEnv(width, height int) []string {
	return []string{
		"COLUMNS=" + itoa(width),
		"LINES=" + itoa(height),
	}
}

func itoa(n int) string {
	// avoid pulling strconv into a one-line helper
	return fmt.Sprintf("%d", n)
}

func TestBuildPreviewCommand_FileSubstitutesEnvBeforeShellEscape(t *testing.T) {
	cfg := &config.PreviewConfig{
		FileCommand: []string{"jj", "diff", "--", "$file"},
		Env: map[string]string{
			"DFT_WIDTH": "$preview_width",
			"X":         "$file",
		},
	}
	item := common.SelectedFile{
		ChangeId: "abc",
		CommitId: "deadbeef",
		File:     "path with space.txt",
	}

	args, env := buildPreviewCommand(item, cfg, "main", 80, 24)

	// $file is shell-escaped in args but raw in env.
	assert.Contains(t, env, "COLUMNS=80")
	assert.Contains(t, env, "LINES=24")
	assert.Contains(t, env, "DFT_WIDTH=80")
	assert.Contains(t, env, "X=path with space.txt")
	// args got the escaped form.
	assert.Equal(t, []string{"jj", "diff", "--", jj.EscapeFileName("path with space.txt")}, args)
}

func TestBuildPreviewCommand_RevisionLeavesUnknownFilePlaceholder(t *testing.T) {
	cfg := &config.PreviewConfig{
		RevisionCommand: []string{"jj", "show", "$change_id"},
		Env:             map[string]string{"X": "$file"},
	}
	item := common.SelectedRevision{ChangeId: "abc", CommitId: "deadbeef"}

	_, env := buildPreviewCommand(item, cfg, "main", 80, 24)

	assert.Contains(t, env, "X=$file")
}

func TestBuildPreviewCommand_NilEnv(t *testing.T) {
	cfg := &config.PreviewConfig{
		RevisionCommand: []string{"jj", "show"},
	}
	item := common.SelectedRevision{ChangeId: "abc", CommitId: "deadbeef"}

	_, env := buildPreviewCommand(item, cfg, "main", 80, 24)

	assert.Equal(t, baseEnv(80, 24), env)
}

func TestBuildPreviewCommand_EmptyEnv(t *testing.T) {
	cfg := &config.PreviewConfig{
		RevisionCommand: []string{"jj", "show"},
		Env:             map[string]string{},
	}
	item := common.SelectedRevision{ChangeId: "abc", CommitId: "deadbeef"}

	_, env := buildPreviewCommand(item, cfg, "main", 80, 24)

	assert.Equal(t, baseEnv(80, 24), env)
}

func TestBuildPreviewCommand_UserOverrideWinsLast(t *testing.T) {
	cfg := &config.PreviewConfig{
		RevisionCommand: []string{"jj", "show"},
		Env:             map[string]string{"COLUMNS": "999"},
	}
	item := common.SelectedRevision{ChangeId: "abc", CommitId: "deadbeef"}

	_, env := buildPreviewCommand(item, cfg, "main", 80, 24)

	// Both entries are present and the user override comes after the base.
	require.Len(t, env, 3)
	assert.Equal(t, "COLUMNS=80", env[0])
	assert.Equal(t, "LINES=24", env[1])
	assert.Equal(t, "COLUMNS=999", env[2])
}

func TestBuildPreviewCommand_AllSelectionTypesSubstituteEnv(t *testing.T) {
	cfg := &config.PreviewConfig{
		FileCommand:     []string{"a"},
		RevisionCommand: []string{"b"},
		EvologCommand:   []string{"c"},
		OplogCommand:    []string{"d"},
		Env:             map[string]string{"W": "$preview_width"},
	}

	cases := []struct {
		name string
		item common.SelectedItem
	}{
		{"file", common.SelectedFile{ChangeId: "x", CommitId: "y", File: "f"}},
		{"revision", common.SelectedRevision{ChangeId: "x", CommitId: "y"}},
		{"commit", common.SelectedCommit{CommitId: "y"}},
		{"operation", common.SelectedOperation{OperationId: "op"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, env := buildPreviewCommand(tc.item, cfg, "main", 80, 24)
			assert.Contains(t, env, "W=80")
		})
	}
}
