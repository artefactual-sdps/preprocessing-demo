package activities

import (
	"context"

	"github.com/artefactual-sdps/preprocessing-demo/internal/premis"
)

const AddPREMISEventName = "add-premis-event"

type (
	AddPREMISEventParams struct {
		PREMISFilePath string
		Agent          premis.Agent
		Summary        premis.EventSummary
	}

	AddPREMISEventResult struct{}

	AddPREMISEventActivity struct{}
)

func NewAddPREMISEvent() *AddPREMISEventActivity {
	return &AddPREMISEventActivity{}
}

func (a *AddPREMISEventActivity) Execute(
	ctx context.Context,
	params *AddPREMISEventParams,
) (*AddPREMISEventResult, error) {
	doc, err := premis.ParseOrInitialize(params.PREMISFilePath)
	if err != nil {
		return nil, err
	}

	err = premis.AppendEventXMLForEachObject(doc, params.Summary, params.Agent)
	if err != nil {
		return nil, err
	}

	err = premis.WriteIndentedToFile(doc, params.PREMISFilePath)
	if err != nil {
		return nil, err
	}

	return &AddPREMISEventResult{}, nil
}
