package activities

import (
	"context"

	"github.com/artefactual-sdps/preprocessing-demo/internal/premis"
)

const AddPREMISEventName = "add-premis-event"

type AddPREMISEventParams struct {
	PREMISFilePath string
	Agent          premis.Agent
	Summary        premis.EventSummary
	Failures       []string
}

type AddPREMISEventResult struct{}

type AddPREMISEventActivity struct{}

func NewAddPREMISEvent() *AddPREMISEventActivity {
	return &AddPREMISEventActivity{}
}

func (md *AddPREMISEventActivity) Execute(
	ctx context.Context,
	params *AddPREMISEventParams,
) (*AddPREMISEventResult, error) {
	doc, err := premis.ParseOrInitialize(params.PREMISFilePath)
	if err != nil {
		return nil, err
	}

	params.Summary.Outcome = "valid"
	if params.Failures != nil {
		params.Summary.Outcome = "invalid"
	}

	err = premis.AppendEventXMLForEachObject(doc, params.Summary, params.Agent)
	if err != nil {
		return nil, err
	}

	err = doc.WriteToFile(params.PREMISFilePath)
	if err != nil {
		return nil, err
	}

	return &AddPREMISEventResult{}, nil
}
