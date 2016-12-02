package ansible

import (
	"os"
	"testing"
	"text/template"

	"github.com/stretchr/testify/require"
)

func TestAnsibleTemplate(t *testing.T) {
	t.Parallel()
	ec := &executionCommon{
		NodeName:            "Welcome",
		Operation:           "tosca.interfaces.node.lifecycle.standard.start",
		Artifacts:           map[string]string{"scripts": "my_scripts"},
		OverlayPath:         "/some/local/path",
		VarInputsNames:      []string{"INSTANCE", "PORT"},
		OperationRemotePath: ".janus/path/on/remote",
	}

	e := &executionAnsible{
		PlaybookPath:    "/some/other/path.yml",
		executionCommon: ec,
	}

	tmpl := template.New("execTest")
	tmpl = tmpl.Delims("[[[", "]]]")
	tmpl, err := tmpl.Parse(ansible_playbook)
	require.Nil(t, err)
	err = tmpl.Execute(os.Stdout, e)
	t.Log(err)
	require.Nil(t, err)

}
