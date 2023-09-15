package utils

import (
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfcommon "github.com/argoproj/argo-workflows/v3/workflow/common"
	argoJson "github.com/argoproj/pkg/json"
)

// unmarshalWorkflows unmarshals the input bytes as either json or yaml
func UnmarshalWorkflows(wfBytes []byte, strict bool) ([]wfv1.Workflow, error) {
	var wf wfv1.Workflow
	var jsonOpts []argoJson.JSONOpt
	if strict {
		jsonOpts = append(jsonOpts, argoJson.DisallowUnknownFields)
	}
	err := argoJson.Unmarshal(wfBytes, &wf, jsonOpts...)
	if err == nil {
		return []wfv1.Workflow{wf}, nil
	}
	yamlWfs, err := wfcommon.SplitWorkflowYAMLFile(wfBytes, strict)
	if err == nil {
		return yamlWfs, nil
	}
	return nil, err
}

func GetDefaultWorkflow() (*wfv1.Workflow, error) {
	content, err := ReadFile(getWorkflowTemplatePath())
	if err != nil {
		return nil, err
	}
	wfs, err := UnmarshalWorkflows([]byte(content), true)
	if err != nil {
		return nil, err
	}
	for _, wf := range wfs {
		return &wf, nil
	}
	return nil, nil
}

func GetDefaultWorkflowStr() (string, error) {
	return ReadFile(getWorkflowTemplatePath())
}

func getWorkflowTemplatePath() string {
	return "utils/workflow-template.yaml"
}
