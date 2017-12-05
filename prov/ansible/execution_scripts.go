package ansible

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"github.com/pkg/errors"

	"strings"

	"novaforge.bull.com/starlings-janus/janus/events"
	"novaforge.bull.com/starlings-janus/janus/helper/executil"
	"novaforge.bull.com/starlings-janus/janus/helper/stringutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/tasks"
)

const outputCustomWrapper = `
#!/usr/bin/env bash
# Workaround JSON structures being treated as python objects
# basically it prevent double quotes to be changed into single quotes
# by prefixing the value by a space
# https://stackoverflow.com/questions/31969872/why-ansible-always-replaces-double-quotes-with-single-quotes-in-templates
# We remove this space here. Obviously becomes a reserved keyword janus_escape_workaround
for janus_escape_workaround in [[[StringsJoin .VarInputsNames " "]]] ;
do
  eval "[[ \"\${${janus_escape_workaround}}\" == \" \"* ]] && { export ${janus_escape_workaround}=\${${janus_escape_workaround}:1};}"
done
[[[printf ". $HOME/%s/%s" $.OperationRemotePath .BasePrimary]]]
[[[range $artName, $art := .Outputs -]]]
[[[printf "echo %s,$%s >> $HOME/%s/out.csv" $artName (cut $artName) $.OperationRemotePath]]]
[[[printf "echo $%s" $artName]]]
[[[end]]]
[[[if .HaveOutput]]]
[[[printf "chmod 777 $HOME/%s/out.csv" $.OperationRemotePath]]]
[[[end]]]
`

const shellAnsiblePlaybook = `
- name: Executing script {{ script_to_run }}
  hosts: all
  strategy: free
  tasks:
    - file: path="{{ ansible_env.HOME}}/[[[.OperationRemotePath]]]" state=directory mode=0755
    [[[printf  "- copy: src=\"{{ wrapper_location }}\" dest=\"{{ ansible_env.HOME}}/%s/wrapper.sh\" mode=0744" $.OperationRemotePath]]]
    - copy: src="{{ script_to_run }}" dest="{{ ansible_env.HOME}}/[[[.OperationRemotePath]]]" mode=0744
    [[[ range $artName, $art := .Artifacts -]]]
    [[[printf "- file: path=\"{{ ansible_env.HOME}}/%s/%s\" state=directory mode=0755" $.OperationRemotePath (path $art)]]]
    [[[printf "- copy: src=\"%s/%s\" dest=\"{{ ansible_env.HOME}}/%s/%s\"" $.OverlayPath $art $.OperationRemotePath (path $art)]]]
    [[[end]]]
    [[[printf "- shell: \"/bin/bash -l {{ ansible_env.HOME}}/%s/wrapper.sh\"" $.OperationRemotePath]]]
      environment:
        [[[ range $key, $envInput := .EnvInputs -]]]
        [[[ if (len $envInput.InstanceName) gt 0]]][[[ if (len $envInput.Value) gt 0]]][[[printf  "%s_%s: %q" $envInput.InstanceName $envInput.Name $envInput.Value]]][[[else]]][[[printf  "%s_%s: \"\"" $envInput.InstanceName $envInput.Name]]]
        [[[end]]][[[else]]][[[ if (len $envInput.Value) gt 0]]][[[printf  "%s: %q" $envInput.Name $envInput.Value]]][[[else]]]
        [[[printf  "%s: \"\"" $envInput.Name]]]
        [[[end]]][[[end]]]
        [[[end]]][[[ range $artName, $art := .Artifacts -]]]
        [[[printf "%s: \"{{ ansible_env.HOME}}/%s/%s\"" $artName $.OperationRemotePath $art]]]
        [[[end]]][[[ range $contextK, $contextV := .Context -]]]
        [[[printf "%s: %q" $contextK $contextV]]]
        [[[end]]][[[ range $hostVarIndex, $hostVarValue := .VarInputsNames -]]]
        [[[printf "%s: \" {{%s}}\"" $hostVarValue $hostVarValue]]]
        [[[end]]]
    [[[if .HaveOutput]]]
    [[[printf "- fetch: src={{ ansible_env.HOME}}/%s/out.csv dest={{dest_folder}}/{{ansible_host}}-out.csv flat=yes" $.OperationRemotePath]]]
    [[[end]]]
    [[[if not .KeepOperationRemotePath ]]]
    - file: path="{{ ansible_env.HOME}}/[[[.OperationRemoteBaseDir]]]" state=absent
    [[[end]]]
`

type executionScript struct {
	*executionCommon
}

func cutAfterLastUnderscore(str string) string {
	idx := strings.LastIndex(str, "_")
	return str[:idx]
}

func (e *executionScript) runAnsible(ctx context.Context, retry bool, currentInstance, ansibleRecipePath string) error {
	// Fill log optional fields for log registration
	wfName, _ := tasks.GetTaskData(e.kv, e.taskID, "workflowName")
	logOptFields := events.LogOptionalFields{
		events.WorkFlowID:    wfName,
		events.NodeID:        e.NodeName,
		events.OperationName: stringutil.GetLastElement(e.operation.Name, "."),
		events.InstanceID:    currentInstance,
		events.InterfaceName: stringutil.GetAllExceptLastElement(e.operation.Name, "."),
	}

	var buffer bytes.Buffer
	funcMap := template.FuncMap{
		// The name "path" is what the function will be called in the template text.
		"path":        filepath.Dir,
		"abs":         filepath.Abs,
		"cut":         cutAfterLastUnderscore,
		"StringsJoin": strings.Join,
	}

	tmpl := template.New("execTemplate")
	tmpl = tmpl.Delims("[[[", "]]]")
	tmpl = tmpl.Funcs(funcMap)
	wrapTemplate := template.New("execTemplate")
	wrapTemplate = wrapTemplate.Delims("[[[", "]]]")
	wrapTemplate, err := tmpl.Parse(outputCustomWrapper)
	if err != nil {
		return err
	}
	if err := wrapTemplate.Execute(&buffer, e); err != nil {
		err = errors.Wrap(err, "Failed to Generate wrapper template")
		events.WithOptionalFields(logOptFields).NewLogEntry(events.ERROR, e.deploymentID).RegisterAsString(err.Error())
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(ansibleRecipePath, "wrapper.sh"), buffer.Bytes(), 0664); err != nil {
		err = errors.Wrap(err, "Failed to write playbook file")
		events.WithOptionalFields(logOptFields).NewLogEntry(events.ERROR, e.deploymentID).RegisterAsString(err.Error())
		return err
	}

	buffer.Reset()
	tmpl, err = tmpl.Parse(shellAnsiblePlaybook)
	if err != nil {
		err = errors.Wrap(err, "Failed to Generate ansible playbook")
		events.WithOptionalFields(logOptFields).NewLogEntry(events.ERROR, e.deploymentID).RegisterAsString(err.Error())
		return err
	}
	if err = tmpl.Execute(&buffer, e); err != nil {
		err = errors.Wrap(err, "Failed to Generate ansible playbook template")
		events.WithOptionalFields(logOptFields).NewLogEntry(events.ERROR, e.deploymentID).RegisterAsString(err.Error())
		return err
	}
	if err = ioutil.WriteFile(filepath.Join(ansibleRecipePath, "run.ansible.yml"), buffer.Bytes(), 0664); err != nil {
		err = errors.Wrap(err, "Failed to write playbook file")
		events.WithOptionalFields(logOptFields).NewLogEntry(events.ERROR, e.deploymentID).RegisterAsString(err.Error())
		return err
	}

	scriptPath, err := filepath.Abs(filepath.Join(e.OverlayPath, e.Primary))
	if err != nil {
		return err
	}

	events.WithOptionalFields(logOptFields).NewLogEntry(events.DEBUG, e.deploymentID).RegisterAsString(fmt.Sprintf("Ansible recipe for node %q: executing %q on remote host(s)", e.NodeName, filepath.Base(scriptPath)))
	var cmd *executil.Cmd
	wrapperPath, err := filepath.Abs(ansibleRecipePath)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve script wrapper absolute path")
	}
	cmd = executil.Command(ctx, "ansible-playbook", "-i", "hosts", "run.ansible.yml", "--extra-vars", fmt.Sprintf("script_to_run=%s , wrapper_location=%s/wrapper.sh , dest_folder=%s", scriptPath, wrapperPath, wrapperPath))
	if _, err = os.Stat(filepath.Join(ansibleRecipePath, "run.ansible.retry")); retry && (err == nil || !os.IsNotExist(err)) {
		cmd.Args = append(cmd.Args, "--limit", filepath.Join("@", ansibleRecipePath, "run.ansible.retry"))
	}
	if e.cfg.AnsibleDebugExec {
		cmd.Args = append(cmd.Args, "-vvvvv")
	}
	if e.cfg.AnsibleUseOpenSSH {
		cmd.Args = append(cmd.Args, "-c", "ssh")
	} else {
		cmd.Args = append(cmd.Args, "-c", "paramiko")
	}
	cmd.Dir = ansibleRecipePath
	var outbuf bytes.Buffer
	errbuf := events.NewBufferedLogEntryWriter()
	cmd.Stdout = &outbuf
	cmd.Stderr = errbuf

	errCloseCh := make(chan bool)
	defer close(errCloseCh)

	// Register log entry via error buffer
	events.WithOptionalFields(logOptFields).NewLogEntry(events.ERROR, e.deploymentID).RunBufferedRegistration(errbuf, errCloseCh)

	defer func(buffer *bytes.Buffer) {
		if err := e.logAnsibleOutputInConsul(buffer, logOptFields); err != nil {
			log.Printf("Failed to publish Ansible log %v", err)
			log.Debugf("%+v", err)
		}
	}(&outbuf)
	if err := cmd.Run(); err != nil {
		return e.checkAnsibleRetriableError(err, logOptFields)
	}

	return nil
}
