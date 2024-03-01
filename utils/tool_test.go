package utils

import "testing"

func TestFile(t *testing.T) {
	file, err := NewFile("./", "text.text", true)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	err = file.Write([]byte("hello world/n"))
	if err != nil {
		t.Fatal(err)
	}
	data, err := file.Read()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(data))
}
