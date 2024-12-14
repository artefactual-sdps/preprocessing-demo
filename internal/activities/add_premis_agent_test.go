package activities_test

import (
	"os"
	"testing"

	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/preprocessing-demo/internal/activities"
	"github.com/artefactual-sdps/preprocessing-demo/internal/premis"
)

const expectedPREMISWithAgent = `<?xml version="1.0" encoding="UTF-8"?>
<premis:premis xmlns:premis="http://www.loc.gov/premis/v3" xmlns:xlink="http://www.w3.org/1999/xlink" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.loc.gov/premis/v3 https://www.loc.gov/standards/premis/premis.xsd" version="3.0">
  <premis:agent>
    <premis:agentIdentifier>
      <premis:agentIdentifierType valueURI="http://id.loc.gov/vocabulary/identifiers/local">url</premis:agentIdentifierType>
      <premis:agentIdentifierValue>https://github.com/artefactual-sdps/preprocessing-sfa</premis:agentIdentifierValue>
    </premis:agentIdentifier>
    <premis:agentName>Enduro</premis:agentName>
    <premis:agentType>software</premis:agentType>
  </premis:agent>
</premis:premis>
`

func TestAddPREMISAgent(t *testing.T) {
	t.Parallel()

	// Transfer that's been deleted (for execution expected to fail).
	transferDeleted := fs.NewDir(t, "",
		fs.WithDir("metadata"),
	)

	PREMISFilePathNonExistent := transferDeleted.Join("metadata", "premis.xml")

	transferDeleted.Remove()

	tests := []struct {
		name       string
		params     activities.AddPREMISAgentParams
		result     activities.AddPREMISAgentResult
		wantErr    string
		wantPREMIS string
	}{
		{
			name: "Add PREMIS agent for normal content",
			params: activities.AddPREMISAgentParams{
				PREMISFilePath: fs.NewDir(t, "",
					fs.WithFile("something.txt", "1234567899"),
					fs.WithDir("metadata"),
				).Join("metadata", "premis.xml"),
				Agent: premis.AgentDefault(),
			},
			result:     activities.AddPREMISAgentResult{},
			wantPREMIS: expectedPREMISWithAgent,
		},
		{
			name: "Add PREMIS agent for no content",
			params: activities.AddPREMISAgentParams{
				PREMISFilePath: fs.NewDir(t, "",
					fs.WithDir("metadata"),
				).Join("metadata", "premis.xml"),
				Agent: premis.AgentDefault(),
			},
			result:     activities.AddPREMISAgentResult{},
			wantPREMIS: expectedPREMISWithAgent,
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

			// Compare PREMIS output to what's expected.
			if tt.wantPREMIS != "" {
				b, err := os.ReadFile(tt.params.PREMISFilePath)
				assert.NilError(t, err)
				assert.Equal(t, string(b), tt.wantPREMIS)
			}
		})
	}
}
