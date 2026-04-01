package desktop

import "testing"

func TestParseRclonePromptPlainJSON(t *testing.T) {
	out := []byte(`{"State":"abc","Option":{"Name":"region"}}`)

	prompt, ok := parseRclonePrompt(out)
	if !ok {
		t.Fatalf("expected prompt to be parsed")
	}
	if prompt.State != "abc" {
		t.Fatalf("unexpected state: %q", prompt.State)
	}
	if prompt.Option.Name != "region" {
		t.Fatalf("unexpected option name: %q", prompt.Option.Name)
	}
}

func TestParseRclonePromptWithNoticeLines(t *testing.T) {
	out := []byte("NOTICE: browser login\n" +
		"NOTICE: waiting code\n" +
		`{"State":"state-1","Option":{"Name":"drive_type"}}` + "\n")

	prompt, ok := parseRclonePrompt(out)
	if !ok {
		t.Fatalf("expected prompt to be parsed from mixed output")
	}
	if prompt.State != "state-1" {
		t.Fatalf("unexpected state: %q", prompt.State)
	}
	if prompt.Option.Name != "drive_type" {
		t.Fatalf("unexpected option name: %q", prompt.Option.Name)
	}
}

func TestParseRclonePromptNoJSON(t *testing.T) {
	out := []byte("NOTICE: something happened\nERROR: failed\n")

	if _, ok := parseRclonePrompt(out); ok {
		t.Fatalf("did not expect prompt for non-json output")
	}
}

func TestParseRclonePromptEmbeddedMultilineJSON(t *testing.T) {
	out := []byte("NOTICE: waiting browser auth\n{\n  \"State\": \"st-42\",\n  \"Option\": {\n    \"Name\": \"config_type\"\n  }\n}\nNOTICE: follow up message\n")

	prompt, ok := parseRclonePrompt(out)
	if !ok {
		t.Fatalf("expected prompt to be parsed from embedded multiline json")
	}
	if prompt.State != "st-42" {
		t.Fatalf("unexpected state: %q", prompt.State)
	}
	if prompt.Option.Name != "config_type" {
		t.Fatalf("unexpected option name: %q", prompt.Option.Name)
	}
}

func TestPreferOneDriveDriveIDPrefersPersonal(t *testing.T) {
	choices := []rcloneConfigChoice{
		{Value: "b!site-lib", Help: "SharePoint Document Library"},
		{Value: "b!personal", Help: "OneDrive Personal Root"},
	}

	got, ok := preferOneDriveDriveID(choices)
	if !ok {
		t.Fatalf("expected a preferred drive_id")
	}
	if got != "b!personal" {
		t.Fatalf("unexpected drive_id: %q", got)
	}
}

func TestPreferOneDriveDriveIDAvoidsSharePointWhenPossible(t *testing.T) {
	choices := []rcloneConfigChoice{
		{Value: "b!site-lib", Help: "SharePoint Site"},
		{Value: "b!other", Help: "Another drive"},
	}

	got, ok := preferOneDriveDriveID(choices)
	if !ok {
		t.Fatalf("expected a preferred drive_id")
	}
	if got != "b!other" {
		t.Fatalf("unexpected drive_id: %q", got)
	}
}

func TestIsTransientOneDriveError(t *testing.T) {
	cases := []string{
		"HTTP error 503 (503 Service Unavailable)",
		"code=serviceNotAvailable",
		"request timeout while waiting",
		"HTTP error 429 too many requests",
	}

	for _, item := range cases {
		if !isTransientOneDriveError(item) {
			t.Fatalf("expected transient for: %q", item)
		}
	}

	if isTransientOneDriveError("unable to get drive_id and drive_type") {
		t.Fatalf("did not expect drive_id error to be treated as transient")
	}
}

func TestOneDriveRetryDelay(t *testing.T) {
	if got := oneDriveRetryDelay(0); got != 1e9 {
		t.Fatalf("unexpected delay for attempt 0: %v", got)
	}
	if got := oneDriveRetryDelay(1); got != 2e9 {
		t.Fatalf("unexpected delay for attempt 1: %v", got)
	}
	if got := oneDriveRetryDelay(3); got != 8e9 {
		t.Fatalf("unexpected delay for attempt 3: %v", got)
	}
}

func TestPreferChoiceMatchesHelpText(t *testing.T) {
	choices := []rcloneConfigChoice{
		{Value: "1", Help: "Business"},
		{Value: "2", Help: "Personal account"},
	}

	got, err := preferChoice(choices, "personal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "2" {
		t.Fatalf("unexpected value: %q", got)
	}
}
