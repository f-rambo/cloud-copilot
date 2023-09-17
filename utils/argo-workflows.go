package utils

import (
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfcommon "github.com/argoproj/argo-workflows/v3/workflow/common"
	argoJson "github.com/argoproj/pkg/json"
)

// unmarshalWorkflows unmarshals the input bytes as either json or yaml
func UnmarshalWorkflow(wfStr string, strict bool) (wfv1.Workflow, error) {
	wfBytes := []byte(wfStr)
	var wf wfv1.Workflow
	var jsonOpts []argoJson.JSONOpt
	if strict {
		jsonOpts = append(jsonOpts, argoJson.DisallowUnknownFields)
	}
	err := argoJson.Unmarshal(wfBytes, &wf, jsonOpts...)
	return wf, err
}

func UnmarshalWorkflows(wfStr string, strict bool) ([]wfv1.Workflow, error) {
	wfBytes := []byte(wfStr)
	return wfcommon.SplitWorkflowYAMLFile(wfBytes, strict)
}

func GetDefaultWorkflow() (*wfv1.Workflow, error) {
	content, err := ReadFile(getWorkflowTemplatePath())
	if err != nil {
		return nil, err
	}
	wf, err := UnmarshalWorkflow(content, true)
	return &wf, err
}

func GetDefaultWorkflowStr() (string, error) {
	return ReadFile(getWorkflowTemplatePath())
}

func getWorkflowTemplatePath() string {
	return "utils/workflow-template.yaml"
}
