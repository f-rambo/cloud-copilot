package operatorapp

import (
	"context"
	"testing"

	operatoroceaniov1alpha1 "github.com/f-rambo/operatorapp/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	// 创建app
	appObj := &operatoroceaniov1alpha1.App{}
	appObj.ObjectMeta = metav1.ObjectMeta{
		Name:      "redis",
		Namespace: "default",
	}
	appObj.TypeMeta = metav1.TypeMeta{
		APIVersion: "operator.ocean.io/v1alpha1",
		Kind:       "App",
	}
	appObj.Spec = operatoroceaniov1alpha1.AppSpec{
		AppChart: operatoroceaniov1alpha1.AppChart{
			Enable:    true,
			RepoName:  "bitnami",
			RepoURL:   "https://charts.bitnami.com/bitnami",
			ChartName: "bitnami/redis",
			Version:   "18.0.0",
			Config:    "",
			Secret:    "",
		},
	}
	resApp := &operatoroceaniov1alpha1.App{}
	appclient, err := NewForConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	resApp, err = appclient.Apps(appObj.Namespace).Create(context.Background(), appObj)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok", resApp.Name)
}
