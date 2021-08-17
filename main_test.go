package main

import "testing"

func TestGetVersion(t *testing.T) {
	//t.Error("foobar")
	cases := []struct {
		in []string
		want string
	}{
		{[]string{"0.14.3"}, "0.14.3"},
	}
	for _, c := range cases {
		got := GetVersion(c.in)
		if got != c.want {
			//t.Error("GetVersion(%q) == %q, want %q", c.in, got, c.want)
			t.Error("foo")
		}
	}
}
