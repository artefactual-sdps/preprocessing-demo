package activities

import (
	"context"

	"github.com/artefactual-sdps/preprocessing-demo/internal/premis"
)

const AddPREMISAgentName = "add-premis-agent"

type (
	AddPREMISAgentParams struct {
		PREMISFilePath string
		Agent          premis.Agent
	}

	AddPREMISAgentResult struct{}

	AddPREMISAgentActivity struct{}
)

func NewAddPREMISAgent() *AddPREMISAgentActivity {
	return &AddPREMISAgentActivity{}
}

func (md *AddPREMISAgentActivity) Execute(
	ctx context.Context,
	params *AddPREMISAgentParams,
) (*AddPREMISAgentResult, error) {
	doc, err := premis.ParseOrInitialize(params.PREMISFilePath)
	if err != nil {
		return nil, err
	}

	err = premis.AppendAgentXML(doc, params.Agent)
	if err != nil {
		return nil, err
	}

	err = premis.WriteIndentedToFile(doc, params.PREMISFilePath)
	if err != nil {
		return nil, err
	}

	return &AddPREMISAgentResult{}, nil
}
