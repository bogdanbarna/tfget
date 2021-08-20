package main

import "testing"

// must run "go test <some other stuff> -vet=off"
// otherwise it fails on formatting
// see https://stackoverflow.com/a/57696603/3431041

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
	// TODO catch errors/log.Fatal() for invalid versions
	/*
		{"foobar"},
		{".314"},
		{"----"},
		{"10"},
	*/
}
