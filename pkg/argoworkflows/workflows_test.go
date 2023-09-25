package argoworkflows

import (
	"context"
	"testing"

	"github.com/f-rambo/ocean/utils"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func TestK(t *testing.T) {
	cfg, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		// 集群内连接
		cfg, err = rest.InClusterConfig()
		if err != nil {
			t.Fatal(err)
		}
	}
	hello := `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: whalesay
  templates:
    - name: whalesay
      container:
        image: docker/whalesay
        command: [cowsay]
        args: ["hello world"]
        resources:
          limits:
            memory: 32Mi
            cpu: 100m`
	wfObj, err := utils.UnmarshalWorkflow(hello, false)
	if err != nil {
		t.Fatal(err)
	}
	wfclient, err := NewForConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	resWf, err := wfclient.Workflows(wfObj.Namespace).Create(context.Background(), &wfObj)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok", resWf.Name)
}
