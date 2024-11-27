package activities_test

import (
	"testing"

	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/preprocessing-demo/internal/activities"
	"github.com/artefactual-sdps/preprocessing-demo/internal/premis"
)

func TestAddPREMISAgent(t *testing.T) {
	t.Parallel()

	// Transfer with one file (for execution expected to work).
	transferOneFile := fs.NewDir(t, "",
		fs.WithFile("something.txt", "1234567899"),
		fs.WithDir("metadata"),
	)

	PREMISFilePathNormal := transferOneFile.Join("metadata", "premis.xml")

	// Transfer with no files (for execution expected to work).
	transferNoFiles := fs.NewDir(t, "",
		fs.WithDir("metadata"),
	)

	PREMISFilePathNoFiles := transferNoFiles.Join("metadata", "premis.xml")

	// Transfer that's been deleted (for execution expected to fail).
	transferDeleted := fs.NewDir(t, "",
		fs.WithDir("metadata"),
	)

	PREMISFilePathNonExistent := transferDeleted.Join("metadata", "premis.xml")

	transferDeleted.Remove()

	tests := []struct {
		name    string
		params  activities.AddPREMISAgentParams
		result  activities.AddPREMISAgentResult
		wantErr string
	}{
		{
			name: "Add PREMIS agent for normal content",
			params: activities.AddPREMISAgentParams{
				PREMISFilePath: PREMISFilePathNormal,
				Agent:          premis.AgentDefault(),
			},
			result: activities.AddPREMISAgentResult{},
		},
		{
			name: "Add PREMIS agent for no content",
			params: activities.AddPREMISAgentParams{
				PREMISFilePath: PREMISFilePathNoFiles,
				Agent:          premis.AgentDefault(),
			},
			result: activities.AddPREMISAgentResult{},
		},
		{
			name: "Add PREMIS agent for bad path",
			params: activities.AddPREMISAgentParams{
				PREMISFilePath: PREMISFilePathNonExistent,
				Agent:          premis.AgentDefault(),
			},
			result:  activities.AddPREMISAgentResult{},
			wantErr: "no such file or directory",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := &temporalsdk_testsuite.WorkflowTestSuite{}
			env := ts.NewTestActivityEnvironment()
			env.RegisterActivityWithOptions(
				activities.NewAddPREMISAgent().Execute,
				temporalsdk_activity.RegisterOptions{Name: activities.AddPREMISAgentName},
			)

			var res activities.AddPREMISAgentResult
			future, err := env.ExecuteActivity(activities.AddPREMISAgentName, tt.params)

			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("error is nil, expecting: %q", tt.wantErr)
				} else {
					assert.ErrorContains(t, err, tt.wantErr)
				}

				return
			}

			assert.NilError(t, err)

			future.Get(&res)
			assert.NilError(t, err)
			assert.DeepEqual(t, res, tt.result)

			_, err = premis.ParseFile(tt.params.PREMISFilePath)
			assert.NilError(t, err)
		})
	}
}
