package workflow_test

import (
	"crypto/rand"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/artefactual-sdps/temporal-activities/bagcreate"
	"github.com/artefactual-sdps/temporal-activities/ffvalidate"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	temporalsdk_worker "go.temporal.io/sdk/worker"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/preprocessing-demo/internal/activities"
	"github.com/artefactual-sdps/preprocessing-demo/internal/config"
	"github.com/artefactual-sdps/preprocessing-demo/internal/enums"
	"github.com/artefactual-sdps/preprocessing-demo/internal/eventlog"
	"github.com/artefactual-sdps/preprocessing-demo/internal/workflow"
)

const allowedFormatsCSV = `Format name,Pronom PUID
text,x-fmt/16
text,x-fmt/21
text,x-fmt/22
text,x-fmt/62
text,x-fmt/111
text,x-fmt/282
text,x-fmt/283
PDF/A,fmt/95
PDF/A,fmt/354
PDF/A,fmt/476
PDF/A,fmt/477
PDF/A,fmt/478
CSV,x-fmt/18
SIARD,fmt/161
SIARD,fmt/1196
SIARD,fmt/1777
TIFF,fmt/353
JPEG 2000,x-fmt/392
WAVE,fmt/1
WAVE,fmt/2
WAVE,fmt/6
WAVE,fmt/141
FFV1,fmt/569
MPEG-4,fmt/199
XML/XSD,fmt/101
XML/XSD,x-fmt/280
INTERLIS,fmt/1014
INTERLIS,fmt/1012
INTERLIS,fmt/654
INTERLIS,fmt/1013
INTERLIS,fmt/1011
INTERLIS,fmt/653`

type PreprocessingTestSuite struct {
	suite.Suite
	temporalsdk_testsuite.WorkflowTestSuite

	env      *temporalsdk_testsuite.TestWorkflowEnvironment
	workflow *workflow.PreprocessingWorkflow
	testDir  string
}

func (s *PreprocessingTestSuite) SetupTest(cfg config.Configuration) {
	s.env = s.NewTestWorkflowEnvironment()
	s.env.SetWorkerOptions(temporalsdk_worker.Options{EnableSessionWorker: true})
	s.testDir = s.T().TempDir()

	// Register activities.
	s.env.RegisterActivityWithOptions(
		ffvalidate.New(cfg.FileFormat).Execute,
		temporalsdk_activity.RegisterOptions{Name: ffvalidate.Name},
	)
	s.env.RegisterActivityWithOptions(
		bagcreate.New(cfg.Bagit).Execute,
		temporalsdk_activity.RegisterOptions{Name: bagcreate.Name},
	)
	s.env.RegisterActivityWithOptions(
		activities.NewAddPREMISAgent().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.AddPREMISAgentName},
	)
	s.env.RegisterActivityWithOptions(
		activities.NewAddPREMISEvent().Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.AddPREMISEventName},
	)
	s.env.RegisterActivityWithOptions(
		activities.NewAddPREMISObjects(rand.Reader).Execute,
		temporalsdk_activity.RegisterOptions{Name: activities.AddPREMISObjectsName},
	)

	s.workflow = workflow.NewPreprocessingWorkflow(s.testDir)
}

func (s *PreprocessingTestSuite) AfterTest(suiteName, testName string) {
	s.env.AssertExpectations(s.T())
}

func TestPreprocessingWorkflow(t *testing.T) {
	suite.Run(t, new(PreprocessingTestSuite))
}

func (s *PreprocessingTestSuite) TestSuccess() {
	transferFiles := fs.NewDir(s.T(), "",
		fs.WithFile("allowed_file_formats.csv", allowedFormatsCSV),
	)

	relPath := transferFiles.Path() // "transfer"

	s.SetupTest(config.Configuration{
		FileFormat: ffvalidate.Config{
			AllowlistPath: transferFiles.Path() + "/allowed_file_formats.csv",
		},
	})
	sessionCtx := mock.AnythingOfType("*context.timerCtx")

	// Mock activities.
	s.env.OnActivity(
		ffvalidate.Name,
		sessionCtx,
		&ffvalidate.Params{Path: filepath.Join(s.testDir, relPath)},
	).Return(
		&ffvalidate.Result{}, nil,
	)

	s.env.OnActivity(
		bagcreate.Name,
		sessionCtx,
		&bagcreate.Params{SourcePath: filepath.Join(s.testDir, relPath)},
	).Return(
		&bagcreate.Result{BagPath: filepath.Join(s.testDir, relPath)},
		nil,
	)

	s.env.ExecuteWorkflow(
		s.workflow.Execute,
		&workflow.PreprocessingWorkflowParams{RelativePath: relPath},
	)

	s.True(s.env.IsWorkflowCompleted())

	var result workflow.PreprocessingWorkflowResult
	err := s.env.GetWorkflowResult(&result)
	s.NoError(err)
	s.Equal(
		&workflow.PreprocessingWorkflowResult{
			Outcome:      workflow.OutcomeSuccess,
			RelativePath: relPath,
			PreservationTasks: []*eventlog.Event{
				{
					Name:        "Validate SIP file formats",
					Message:     "No disallowed file formats found",
					Outcome:     enums.EventOutcomeSuccess,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
				{
					Name:        "Bag SIP",
					Message:     "SIP has been bagged",
					Outcome:     enums.EventOutcomeSuccess,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
				{
					Name:        "Create premis.xml",
					Message:     "Created a premis.xml and stored in metadata directory",
					Outcome:     enums.EventOutcomeSuccess,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
			},
		},
		&result,
	)
}

func (s *PreprocessingTestSuite) TestNoRelativePathError() {
	s.SetupTest(config.Configuration{})
	s.env.ExecuteWorkflow(
		s.workflow.Execute,
		&workflow.PreprocessingWorkflowParams{},
	)

	s.True(s.env.IsWorkflowCompleted())

	var result workflow.PreprocessingWorkflowResult
	err := s.env.GetWorkflowResult(&result)
	s.ErrorContains(err, "error calling workflow with unexpected inputs")
}

func (s *PreprocessingTestSuite) TestSystemError() {
	relPath := "transfer"
	s.SetupTest(config.Configuration{
		FileFormat: ffvalidate.Config{
			AllowlistPath: "./testdata/allowed_file_formats.csv",
		},
	})
	sessionCtx := mock.AnythingOfType("*context.timerCtx")

	// Mock activities.
	s.env.OnActivity(
		ffvalidate.Name,
		sessionCtx,
		&ffvalidate.Params{Path: filepath.Join(s.testDir, relPath)},
	).Return(
		&ffvalidate.Result{}, nil,
	)

	s.env.OnActivity(
		bagcreate.Name,
		sessionCtx,
		&bagcreate.Params{SourcePath: filepath.Join(s.testDir, relPath)},
	).Return(
		nil,
		fmt.Errorf(
			"bagcreate: failed to open %s: permission denied",
			filepath.Join(s.testDir, relPath),
		),
	)

	s.env.ExecuteWorkflow(
		s.workflow.Execute,
		&workflow.PreprocessingWorkflowParams{RelativePath: relPath},
	)

	s.True(s.env.IsWorkflowCompleted())

	var result workflow.PreprocessingWorkflowResult
	err := s.env.GetWorkflowResult(&result)
	s.NoError(err)
	s.Equal(
		&workflow.PreprocessingWorkflowResult{
			Outcome:      workflow.OutcomeSystemError,
			RelativePath: relPath,
			PreservationTasks: []*eventlog.Event{
				{
					Name:        "Validate SIP file formats",
					Message:     "No disallowed file formats found",
					Outcome:     enums.EventOutcomeSuccess,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
				{
					Name:        "Bag SIP",
					Message:     "System error: bagging has failed",
					Outcome:     enums.EventOutcomeSystemFailure,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
			},
		},
		&result,
	)
}

func (s *PreprocessingTestSuite) TestFFValidationError() {
	relPath := "transfer"
	s.SetupTest(config.Configuration{
		FileFormat: ffvalidate.Config{
			AllowlistPath: "./testdata/allowed_file_formats.csv",
		},
	})
	sessionCtx := mock.AnythingOfType("*context.timerCtx")

	// Mock activities.
	s.env.OnActivity(
		ffvalidate.Name,
		sessionCtx,
		&ffvalidate.Params{Path: filepath.Join(s.testDir, relPath)},
	).Return(
		&ffvalidate.Result{
			Failures: []string{
				`file format "fmt/11" not allowed: "test_transfer/content/content/dir/file1.png"`,
			},
		}, nil,
	)

	s.env.ExecuteWorkflow(
		s.workflow.Execute,
		&workflow.PreprocessingWorkflowParams{RelativePath: relPath},
	)

	s.True(s.env.IsWorkflowCompleted())

	var result workflow.PreprocessingWorkflowResult
	err := s.env.GetWorkflowResult(&result)
	s.NoError(err)
	s.Equal(
		&workflow.PreprocessingWorkflowResult{
			Outcome:      workflow.OutcomeContentError,
			RelativePath: relPath,
			PreservationTasks: []*eventlog.Event{
				{
					Name: "Validate SIP file formats",
					Message: `Content error: file format validation has failed. One or more file formats are not allowed:
file format "fmt/11" not allowed: "test_transfer/content/content/dir/file1.png"`,
					Outcome:     enums.EventOutcomeValidationFailure,
					StartedAt:   s.env.Now().UTC(),
					CompletedAt: s.env.Now().UTC(),
				},
			},
		},
		&result,
	)
}
