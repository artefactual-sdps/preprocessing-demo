package activities_test

import (
	pseudorand "math/rand"
	"os"
	"testing"

	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/preprocessing-demo/internal/activities"
)

const expectedPREMIS = `<?xml version="1.0" encoding="UTF-8"?>
<premis:premis xmlns:premis="http://www.loc.gov/premis/v3" xmlns:xlink="http://www.w3.org/1999/xlink" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.loc.gov/premis/v3 https://www.loc.gov/standards/premis/premis.xsd" version="3.0">
  <premis:object xsi:type="premis:file">
    <premis:objectIdentifier>
      <premis:objectIdentifierType>UUID</premis:objectIdentifierType>
      <premis:objectIdentifierValue>52fdfc07-2182-454f-963f-5f0f9a621d72</premis:objectIdentifierValue>
    </premis:objectIdentifier>
    <premis:objectCharacteristics>
      <premis:format>
        <premis:formatDesignation>
          <premis:formatName/>
        </premis:formatDesignation>
      </premis:format>
    </premis:objectCharacteristics>
    <premis:originalName>somefile.txt</premis:originalName>
  </premis:object>
</premis:premis>
`

const expectedPREMISNoFiles = `<?xml version="1.0" encoding="UTF-8"?>
<premis:premis xmlns:premis="http://www.loc.gov/premis/v3" xmlns:xlink="http://www.w3.org/1999/xlink" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.loc.gov/premis/v3 https://www.loc.gov/standards/premis/premis.xsd" version="3.0"/>
`

func TestAddPREMISObjects(t *testing.T) {
	t.Parallel()

	// Test transfer with one file.
	transferOneFile := fs.NewDir(t, "",
		fs.WithFile("somefile.txt", "somestuff"),
	)
	premisFilePathOneFile := transferOneFile.Join("metadata", "premis.xml")

	// Test transfer with no files.
	transferNoFiles := fs.NewDir(t, "")
	premisFilePathNoFiles := transferNoFiles.Join("metadata", "premis.xml")

	tests := []struct {
		name       string
		params     activities.AddPREMISObjectsParams
		result     activities.AddPREMISObjectsResult
		wantPREMIS string
		wantErr    string
	}{
		{
			name: "Add PREMIS objects for transfer with one file",
			params: activities.AddPREMISObjectsParams{
				SIPPath:        transferOneFile.Path(),
				PREMISFilePath: premisFilePathOneFile,
			},
			result:     activities.AddPREMISObjectsResult{},
			wantPREMIS: expectedPREMIS,
		},
		{
			name: "Add PREMIS objects for empty transfer",
			params: activities.AddPREMISObjectsParams{
				SIPPath:        transferNoFiles.Path(),
				PREMISFilePath: premisFilePathNoFiles,
			},
			result:     activities.AddPREMISObjectsResult{},
			wantPREMIS: expectedPREMISNoFiles,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := &temporalsdk_testsuite.WorkflowTestSuite{}
			env := ts.NewTestActivityEnvironment()
			rng := pseudorand.New(pseudorand.NewSource(1)) // #nosec G404
			env.RegisterActivityWithOptions(
				activities.NewAddPREMISObjects(rng).Execute,
				temporalsdk_activity.RegisterOptions{Name: activities.AddPREMISObjectsName},
			)

			var res activities.AddPREMISObjectsResult
			future, err := env.ExecuteActivity(activities.AddPREMISObjectsName, tt.params)

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
			assert.DeepEqual(t, res, tt.result)

			b, err := os.ReadFile(tt.params.PREMISFilePath)
			assert.NilError(t, err)
			assert.Equal(t, string(b), tt.wantPREMIS)
		})
	}
}
