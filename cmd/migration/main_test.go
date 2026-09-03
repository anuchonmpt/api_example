package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseCLIArgs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, env, command string
		argv, args         []string
		wantError          string
	}{
		{name: "local up", argv: []string{"migration", "up"}, command: "up"},
		{name: "environment down", argv: []string{"migration", "staging", "down", "2"}, env: "staging", command: "down", args: []string{"2"}},
		{name: "unsupported", argv: []string{"migration", "destroy"}, wantError: "unsupported"},
		{name: "missing", argv: []string{"migration"}, wantError: "missing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			env, command, args, err := parseCLIArgs(tt.argv)
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("error = %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if env != tt.env || command != tt.command || !reflect.DeepEqual(args, tt.args) {
				t.Fatalf("got (%q, %q, %v), want (%q, %q, %v)", env, command, args, tt.env, tt.command, tt.args)
			}
		})
	}
}

func TestParseDownSteps(t *testing.T) {
	t.Parallel()
	tests := []struct {
		args      []string
		want      int
		wantError bool
	}{
		{args: nil, want: 1}, {args: []string{"3"}, want: 3}, {args: []string{"0"}, wantError: true}, {args: []string{"-1"}, wantError: true}, {args: []string{"x"}, wantError: true},
	}
	for _, tt := range tests {
		got, err := parseDownSteps(tt.args)
		if (err != nil) != tt.wantError || (!tt.wantError && got != tt.want) {
			t.Fatalf("parseDownSteps(%v) = (%d, %v)", tt.args, got, err)
		}
	}
}
