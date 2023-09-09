package restapi

import "testing"

func TestGet(t *testing.T) {
	data, err := GetContentByUrl("https://raw.githubusercontent.com/f-rambo/kubespray/master/inventory/sample/group_vars/k8s_cluster/addons.yml")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(data)
}
