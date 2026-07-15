package main

import "testing"

func TestParsersExposeOnlyGenericAttemptProvenance(t *testing.T) {
	args, flags, err := parseExecArgs([]string{"init", "u", "001", "--attempt", "a1", "--executor", "custom", "--vendor", "vendor", "--model", "model", "--locality", "local", "--execution-ref", "run-1"})
	if err != nil || len(args) != 3 || flags.Attempt != "a1" || flags.Executor != "custom" || flags.Vendor != "vendor" || flags.ExecutionRef != "run-1" {
		t.Fatalf("args=%v flags=%#v err=%v", args, flags, err)
	}
}

func TestVerifyJSONFlag(t *testing.T) {
	args, flags, err := parseVerifyArgs([]string{"u", "001", "--attempt", "a1", "--json"})
	if err != nil || len(args) != 2 || flags.Attempt != "a1" || !flags.JSON {
		t.Fatalf("args=%v flags=%#v err=%v", args, flags, err)
	}
}
