package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/artefactual-sdps/temporal-activities/bagcreate"
	"github.com/artefactual-sdps/temporal-activities/ffvalidate"
	"go.artefactual.dev/tools/temporal"
	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"

	"github.com/artefactual-sdps/preprocessing-demo/internal/activities"
	"github.com/artefactual-sdps/preprocessing-demo/internal/enums"
	"github.com/artefactual-sdps/preprocessing-demo/internal/eventlog"
	"github.com/artefactual-sdps/preprocessing-demo/internal/premis"
)

type Outcome int

const (
	OutcomeSuccess Outcome = iota
	OutcomeSystemError
	OutcomeContentError
)

type PreprocessingWorkflowParams struct {
	RelativePath string
}

type PreprocessingWorkflowResult struct {
	Outcome           Outcome
	RelativePath      string
	PreservationTasks []*eventlog.Event
}

func (r *PreprocessingWorkflowResult) newEvent(ctx temporalsdk_workflow.Context, name string) *eventlog.Event {
	ev := eventlog.NewEvent(temporalsdk_workflow.Now(ctx), name)
	r.PreservationTasks = append(r.PreservationTasks, ev)

	return ev
}

func (r *PreprocessingWorkflowResult) validationError(
	ctx temporalsdk_workflow.Context,
	ev *eventlog.Event,
	msg string,
	failures []string,
) *PreprocessingWorkflowResult {
	r.Outcome = OutcomeContentError
	ev.Complete(
		temporalsdk_workflow.Now(ctx),
		enums.EventOutcomeValidationFailure,
		"Content error: %s:\n%s",
		msg,
		strings.Join(failures, "\n"),
	)

	return r
}

func (r *PreprocessingWorkflowResult) systemError(
	ctx temporalsdk_workflow.Context,
	err error,
	ev *eventlog.Event,
	msg string,
) *PreprocessingWorkflowResult {
	logger := temporalsdk_workflow.GetLogger(ctx)
	logger.Error("System error", "message", err.Error())

	ev.Complete(
		temporalsdk_workflow.Now(ctx),
		enums.EventOutcomeSystemFailure,
		"System error: %s",
		msg,
	)
	r.Outcome = OutcomeSystemError

	return r
}

type PreprocessingWorkflow struct {
	sharedPath string
}

func NewPreprocessingWorkflow(sharedPath string) *PreprocessingWorkflow {
	return &PreprocessingWorkflow{
		sharedPath: sharedPath,
	}
}

func (w *PreprocessingWorkflow) Execute(
	ctx temporalsdk_workflow.Context,
	params *PreprocessingWorkflowParams,
) (*PreprocessingWorkflowResult, error) {
	var e error
	result := &PreprocessingWorkflowResult{}

	logger := temporalsdk_workflow.GetLogger(ctx)
	logger.Debug("PreprocessingWorkflow workflow running!", "params", params)

	if params == nil || params.RelativePath == "" {
		e = temporal.NewNonRetryableError(fmt.Errorf("error calling workflow with unexpected inputs"))
		return nil, e
	}
	result.RelativePath = params.RelativePath

	// Validate file formats.
	ev := result.newEvent(ctx, "Validate SIP file formats")
	var validateFileFormat ffvalidate.Result
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		ffvalidate.Name,
		&ffvalidate.Params{Path: filepath.Join(w.sharedPath, params.RelativePath)},
	).Get(ctx, &validateFileFormat)
	if e != nil {
		result.systemError(ctx, e, ev, "file format validation has failed")
		return result, nil
	}
	if validateFileFormat.Failures != nil {
		result.validationError(
			ctx,
			ev,
			"file format validation has failed. One or more file formats are not allowed",
			validateFileFormat.Failures,
		)
	} else {
		ev.Succeed(temporalsdk_workflow.Now(ctx), "No disallowed file formats found")
	}

	// Stop here if there are validation errors.
	if result.Outcome == OutcomeContentError {
		return result, nil
	}

	// Bag the SIP for Enduro processing.
	ev = result.newEvent(ctx, "Bag SIP")
	var createBag bagcreate.Result
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		bagcreate.Name,
		&bagcreate.Params{
			SourcePath: filepath.Join(w.sharedPath, params.RelativePath),
		},
	).Get(ctx, &createBag)
	if e != nil {
		return result.systemError(ctx, e, ev, "bagging has failed"), nil
	}
	ev.Succeed(temporalsdk_workflow.Now(ctx), "SIP has been bagged")

	// Write PREMIS XML.
	ev = result.newEvent(ctx, "Create premis.xml")
	if e = writePREMISFile(ctx, filepath.Join(w.sharedPath, params.RelativePath)); e != nil {
		result.systemError(ctx, e, ev, "premis.xml creation has failed")
	} else {
		ev.Succeed(temporalsdk_workflow.Now(ctx), "Created a premis.xml and stored in metadata directory")
	}

	return result, nil
}

func withLocalActOpts(ctx temporalsdk_workflow.Context) temporalsdk_workflow.Context {
	return temporalsdk_workflow.WithActivityOptions(
		ctx,
		temporalsdk_workflow.ActivityOptions{
			ScheduleToCloseTimeout: 5 * time.Minute,
			RetryPolicy: &temporalsdk_temporal.RetryPolicy{
				MaximumAttempts: 1,
			},
		},
	)
}

func writePREMISFile(ctx temporalsdk_workflow.Context, sipPath string) error {
	var e error
	metadataPath := filepath.Join(sipPath, "metadata")
	premisFilePath := filepath.Join(metadataPath, "premis.xml")

	// Add metadata directory if it doesn't exist.
	if _, err := os.Stat(metadataPath); err != nil {
		err = os.MkdirAll(metadataPath, 0o750)
		if err != nil {
			return err
		}
	}

	// Add PREMIS objects.
	var addPREMISObjects activities.AddPREMISObjectsResult
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.AddPREMISObjectsName,
		&activities.AddPREMISObjectsParams{
			SIPPath:        sipPath,
			PREMISFilePath: premisFilePath,
		},
	).Get(ctx, &addPREMISObjects)
	if e != nil {
		return e
	}

	// Add PREMIS event noting validate SIP file formats result.
	validateStructureOutcomeDetail := "It's good"

	var addPREMISEvent activities.AddPREMISEventResult
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.AddPREMISEventName,
		&activities.AddPREMISEventParams{
			PREMISFilePath: premisFilePath,
			Agent:          premis.AgentDefault(),
			Type:           "validation",
			Detail:         "name=\"Validate SIP file formats\"",
			OutcomeDetail:  validateStructureOutcomeDetail,
			Failures:       nil,
		},
	).Get(ctx, &addPREMISEvent)
	if e != nil {
		return e
	}

	// Add PREMIS events for SIP bagging.
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.AddPREMISEventName,
		&activities.AddPREMISEventParams{
			PREMISFilePath: premisFilePath,
			Agent:          premis.AgentDefault(),
			Type:           "validation",
			Detail:         "name=\"Bag SIP\"",
			OutcomeDetail:  "Format allowed",
			Failures:       nil,
		},
	).Get(ctx, &addPREMISEvent)
	if e != nil {
		return e
	}

	// Add Enduro PREMIS agent.
	var addPREMISAgent activities.AddPREMISAgentResult
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		activities.AddPREMISAgentName,
		&activities.AddPREMISAgentParams{
			PREMISFilePath: premisFilePath,
			Agent:          premis.AgentDefault(),
		},
	).Get(ctx, &addPREMISAgent)
	if e != nil {
		return e
	}

	return nil
}
