package commands

import (
	"bytes"
	"testing"

	"github.com/rogerwelin/cfnctl/internal/mock"
	"github.com/rogerwelin/cfnctl/pkg/client"
)

func TestPlan(t *testing.T) {

	expectedStr := "\nCfnctl will perform the following actions:\n\n\nPlan: 0 to add, 0 to change, 0 to destroy\n\n"

	svc := mock.NewMockAPI()
	buf := &bytes.Buffer{}

	ctl := client.New(
		client.WithSvc(svc),
		client.WithStackName("stack"),
		client.WithChangesetName("change-stack"),
		client.WithTemplatePath("testdata/template.yaml"),
		client.WithAutoApprove(true),
		client.WithOutput(buf),
	)

	_, err := Plan(ctl, false)
	if err != nil {
		t.Errorf("Expected err to be nil but got: %v", err)
	}

	if buf.String() != expectedStr {
		t.Errorf("Expected str:\n%s but got:\n %s", expectedStr, buf.String())
	}
}
