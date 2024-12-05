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

const expectedPREMISWithSuccessfulEvent = `<?xml version="1.0" encoding="UTF-8"?>
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
    <premis:linkingEventIdentifier>
      <premis:linkingEventIdentifierType/>
      <premis:linkingEventIdentifierValue/>
    </premis:linkingEventIdentifier>
  </premis:object>
  <premis:event>
    <premis:eventIdentifier>
      <premis:eventIdentifierType/>
      <premis:eventIdentifierValue/>
    </premis:eventIdentifier>
    <premis:eventType>someActivity</premis:eventType>
    <premis:eventDateTime/>
    <premis:eventDetailInformation>
      <premis:eventDetail/>
    </premis:eventDetailInformation>
    <premis:eventOutcomeInformation>
      <premis:eventOutcome>valid</premis:eventOutcome>
    </premis:eventOutcomeInformation>
    <premis:linkingAgentIdentifier>
      <premis:linkingAgentIdentifierType valueURI="http://id.loc.gov/vocabulary/identifiers/local">url</premis:linkingAgentIdentifierType>
      <premis:linkingAgentIdentifierValue>https://github.com/artefactual-sdps/preprocessing-sfa</premis:linkingAgentIdentifierValue>
    </premis:linkingAgentIdentifier>
  </premis:event>
</premis:premis>
`

const expectedPREMISWithUnsuccessfulEvent = `<?xml version="1.0" encoding="UTF-8"?>
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
    <premis:linkingEventIdentifier>
      <premis:linkingEventIdentifierType/>
      <premis:linkingEventIdentifierValue/>
    </premis:linkingEventIdentifier>
  </premis:object>
  <premis:event>
    <premis:eventIdentifier>
      <premis:eventIdentifierType/>
      <premis:eventIdentifierValue/>
    </premis:eventIdentifier>
    <premis:eventType>someActivity</premis:eventType>
    <premis:eventDateTime/>
    <premis:eventDetailInformation>
      <premis:eventDetail/>
    </premis:eventDetailInformation>
    <premis:eventOutcomeInformation>
      <premis:eventOutcome>invalid</premis:eventOutcome>
    </premis:eventOutcomeInformation>
    <premis:linkingAgentIdentifier>
      <premis:linkingAgentIdentifierType valueURI="http://id.loc.gov/vocabulary/identifiers/local">url</premis:linkingAgentIdentifierType>
      <premis:linkingAgentIdentifierValue>https://github.com/artefactual-sdps/preprocessing-sfa</premis:linkingAgentIdentifierValue>
    </premis:linkingAgentIdentifier>
  </premis:event>
</premis:premis>
`

func TestAddPREMISEvent(t *testing.T) {
	t.Parallel()

	// Normal execution with no failures (for execution expected to work).
	PREMISFilePathNormalNoFailures := fs.NewFile(t, "premis.xml",
		fs.WithContent(expectedPREMISWithFile),
	).Path()

	// Normal execution with failures (for execution expected to work).
	PREMISFilePathNormalWithFailures := fs.NewFile(t, "premis.xml",
		fs.WithContent(expectedPREMISWithFile),
	).Path()

	// Creation of PREMIS file in existing directory (for execution expected to work).
	transferNoFiles := fs.NewDir(t, "",
		fs.WithDir("metadata"),
	)

	PREMISFilePathNoFiles := transferNoFiles.Join("metadata", "premis.xml")

	// Creation of PREMIS file in non-existing directory (for execution expected to fail).
	transferDeleted := fs.NewDir(t, "",
		fs.WithDir("metadata"),
	)

	PREMISFilePathNonExistent := transferDeleted.Join("metadata", "premis.xml")

	transferDeleted.Remove()

	// No failures.
	var noFailures []string

	// Failure.
	var failures []string
	failures = append(failures, "some failure")

	tests := []struct {
		name       string
		params     activities.AddPREMISEventParams
		result     activities.AddPREMISEventResult
		wantErr    string
		wantPREMIS string
	}{
		{
			name: "Add PREMIS event for normal content with no failures",
			params: activities.AddPREMISEventParams{
				PREMISFilePath: PREMISFilePathNormalNoFailures,
				Agent:          premis.AgentDefault(),
				Summary: premis.EventSummary{
					Type: "someActivity",
				},
				Failures: noFailures,
			},
			result:     activities.AddPREMISEventResult{},
			wantPREMIS: expectedPREMISWithSuccessfulEvent,
		},
		{
			name: "Add PREMIS event for normal content with failures",
			params: activities.AddPREMISEventParams{
				PREMISFilePath: PREMISFilePathNormalWithFailures,
				Agent:          premis.AgentDefault(),
				Summary: premis.EventSummary{
					Type: "someActivity",
				},
				Failures: failures,
			},
			result:     activities.AddPREMISEventResult{},
			wantPREMIS: expectedPREMISWithUnsuccessfulEvent,
		},
		{
			name: "Add PREMIS event for no content",
			params: activities.AddPREMISEventParams{
				PREMISFilePath: PREMISFilePathNoFiles,
				Agent:          premis.AgentDefault(),
				Summary: premis.EventSummary{
					Type: "someActivity",
				},
				Failures: noFailures,
			},
			result:     activities.AddPREMISEventResult{},
			wantPREMIS: premis.EmptyXML,
		},
		{
			name: "Add PREMIS event for bad path",
			params: activities.AddPREMISEventParams{
				PREMISFilePath: PREMISFilePathNonExistent,
				Agent:          premis.AgentDefault(),
				Summary: premis.EventSummary{
					Type: "someActivity",
				},
				Failures: noFailures,
			},
			result:  activities.AddPREMISEventResult{},
			wantErr: "no such file or directory",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := &temporalsdk_testsuite.WorkflowTestSuite{}
			env := ts.NewTestActivityEnvironment()
			env.RegisterActivityWithOptions(
				activities.NewAddPREMISEvent().Execute,
				temporalsdk_activity.RegisterOptions{Name: activities.AddPREMISEventName},
			)

			var res activities.AddPREMISEventResult
			future, err := env.ExecuteActivity(activities.AddPREMISEventName, tt.params)

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
			assert.NilError(t, err)

			// Compare PREMIS output to what's expected.
			if tt.wantPREMIS != "" {
				doc, err := premis.ParseFile(tt.params.PREMISFilePath)
				if err != nil {
					t.Errorf("error parsing premis xml")
				}

				xml, err := doc.WriteToString()
				if err != nil {
					t.Errorf("error writing xml too string")
				}

				assert.Equal(t, xml, tt.wantPREMIS)
			}
		})
	}
}
