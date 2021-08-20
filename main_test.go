package main

import (
	"archive/zip"
	"bytes"
	"testing"
)

// https://golang.org/doc/code#Testing

// must run "go test <some other stuff> -vet=off"
// otherwise it fails on formatting
// see https://stackoverflow.com/a/57696603/3431041

//https://ieftimov.com/post/testing-in-go-failing-tests/

func TestDeterminVersion(t *testing.T) {
	versions := []string{"1.0.0", "0.14.3", "0.12", "0.6.3"}
	cases := []struct {
		in   string
		want string
	}{
		{"0.14.3", "0.14.3"},
		{"latest", "1.0.0"},
		{"0.12", "0.12"},
	}
	for _, c := range cases {
		got := DetermineVersion(c.in, versions)
		if got != c.want {
			t.Errorf("DetermineVersion(%v) == %v, want %v", c.in, got, c.want)
		} else {
			t.Logf("DetermineVersion(%v) == %v", c.in, got)
		}
	}
	/*
		TODO catch errors/log.Fatal() for invalid versions
		https://stackoverflow.com/questions/30688554/how-to-test-go-function-containing-log-fatal
		{"foobar"},
		{".314"},
		{"----"},
		{"10"},
	*/
}

/*
TODO test these
func UnzipTerraformArchive(fullPath string)
func DownloadTerraform(dirPath string, version_number string)
func ListRemoteVersions() []string
func ListLocal(dirPath string)
func DetermineVersion(cliArg string, versions []string) string
func mkdirLocalCache(dirPath string) string
*/

func TestUnzipTerraformArchive(t *testing.T) {
	// Zip archive
	// https://pkg.go.dev/archive/zip#example-Writer
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	var file = []struct {
		Name, Body string
	}{
		{"foo.txt", "Dummy file"},
	}
	f, err := w.Create(file[0].Name)
	if err != nil {
		t.Error(err)
	}
	_, err = f.Write([]byte(file[0].Body))
	if err != nil {
		t.Error(err)
	}

	err = w.Close()
	if err != nil {
		t.Error(err)
	}
	// Validate Unzip
	//UnzipTerraformArchive(file[0].Name)
}
