package argoworkflows

import (
	"testing"

	"github.com/f-rambo/ocean/utils"
)

func TestK(t *testing.T) {
	file, err := utils.NewFile("./", "workflow-template.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	data, err := file.Read()
	if err != nil {
		t.Fatal(err)
	}
	wf, err := UnmarshalWorkflow(string(data), true)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(wf)
}
