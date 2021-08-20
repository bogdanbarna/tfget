package main

import "testing"

// must run "go test <some other stuff> -vet=off"
// otherwise it fails on formatting
// see https://stackoverflow.com/a/57696603/3431041

func TestGetVersion(t *testing.T) {
	cases := []struct {
		in   []string
		want string
	}{
		{[]string{"0.14.3"}, "0.14.3"},
	}
	for _, c := range cases {
		got := GetVersion(c.in)
		if got != c.want {
			t.Error("GetVersion(%q) == %q, want %q", c.in, got, c.want)
		}
	}
}

func TestValidateVersion(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"latest", true},
		{"0.14.3", true},
		{"1.0.5", true},
		{"0.12", true},
		{"foobar", false},
		{".314", false},
		{"----", false},
		{"10", false},
	}
	for _, c := range cases {
		got := ValidateVersion(c.in)
		if got != c.want {
			t.Error("ValidateVersion(%q) == %t; want %t", c.in, got, c.want)
		}
	}
}
