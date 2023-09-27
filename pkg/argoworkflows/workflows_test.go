package argoworkflows

import (
	"testing"

	"github.com/f-rambo/ocean/utils"
)

func TestK(t *testing.T) {
	tem, _ := utils.ReadFile("workflow-template.yaml")
	wf, err := UnmarshalWorkflow(tem, true)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(wf)
}
