package terraform

import (
	"context"
	"fmt"
	"github.com/hashicorp/consul/api"
	"io"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/prov/terraform/openstack"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Executor interface {
	ProvisionNode(ctx context.Context, deploymentId, nodeName string) error
	DestroyNode(ctx context.Context, deploymentId, nodeName string) error
}

type defaultExecutor struct {
	kv  *api.KV
	cfg config.Configuration
}

func NewExecutor(kv *api.KV, cfg config.Configuration) Executor {
	return &defaultExecutor{kv: kv, cfg: cfg}
}

type BufferedConsulWriter struct {
	kv        *api.KV
	depId     string
	buf       []byte
	completed []byte
	n         int
	io.Writer
}

func NewWriterSize(api *api.KV, depId string) *BufferedConsulWriter {
	return &BufferedConsulWriter{
		buf:   make([]byte, 0),
		kv:    api,
		depId: depId,
	}
}

func (b *BufferedConsulWriter) Write(p []byte) (nn int, err error) {
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *BufferedConsulWriter) Flush() error {
	//fmt.Printf(string(p))
	if len(b.buf) == 0 {
		return nil
	}
	fmt.Printf(string(b.buf))
	reg := regexp.MustCompile(`\x1B\[([0-9]{1,2}(;[0-9]{1,2})?)?[m|K]`)
	out := reg.ReplaceAll(b.buf, []byte(""))
	kv := &api.KVPair{Key: filepath.Join(deployments.DeploymentKVPrefix, b.depId, "logs", log.INFRA_LOG_PREFIX+"__"+time.Now().Format(time.RFC3339Nano)), Value: out}
	_, err := b.kv.Put(kv, nil)
	if err != nil {
		return err
	}
	b.buf = b.buf[:0]
	return nil

}

func (b *BufferedConsulWriter) run(quit chan bool) {
	go func() {
		for {
			select {
			case <-quit:
				return
			case <-time.After(5 * time.Second):
				b.Flush()
			}
		}
	}()
}

func (e *defaultExecutor) ProvisionNode(ctx context.Context, deploymentId, nodeName string) error {

	kvPair, _, err := e.kv.Get(path.Join(deployments.DeploymentKVPrefix, deploymentId, "topology/nodes", nodeName, "type"), nil)
	if err != nil {
		return err
	}
	if kvPair == nil {
		return fmt.Errorf("Type for node '%s' in deployment '%s' not found", nodeName, deploymentId)
	}
	nodeType := string(kvPair.Value)
	infraGenerated := true
	switch {
	case strings.HasPrefix(nodeType, "janus.nodes.openstack."):
		osGenerator := openstack.NewGenerator(e.kv, e.cfg)
		if infraGenerated, err = osGenerator.GenerateTerraformInfraForNode(deploymentId, nodeName); err != nil {
			return err
		}
	default:
		return fmt.Errorf("Unsupported node type '%s' for node '%s' in deployment '%s'", nodeType, nodeName, deploymentId)
	}
	if infraGenerated {
		if err := e.applyInfrastructure(ctx, deploymentId, nodeName); err != nil {
			return err
		}
	}
	return nil
}

func (e *defaultExecutor) DestroyNode(ctx context.Context, deploymentId, nodeName string) error {
	if err := e.destroyInfrastructure(ctx, deploymentId, nodeName); err != nil {
		return err
	}
	return nil
}

func (e *defaultExecutor) applyInfrastructure(ctx context.Context, depId, nodeName string) error {
	log.StoreInConsul(e.kv, depId, "Applying the infrastructure")
	infraPath := filepath.Join("work", "deployments", depId, "infra", nodeName)
	cmd := exec.CommandContext(ctx, "terraform", "apply")
	cmd.Dir = infraPath
	errbuf := NewWriterSize(e.kv, depId)
	out := NewWriterSize(e.kv, depId)
	cmd.Stdout = out
	cmd.Stderr = errbuf

	quit := make(chan bool)
	out.run(quit)
	errbuf.run(quit)

	if err := cmd.Start(); err != nil {
		log.Print(err)
	}

	err := cmd.Wait()
	quit <- true

	return err

}

func (e *defaultExecutor) destroyInfrastructure(ctx context.Context, depId, nodeName string) error {
	nodePath := path.Join(deployments.DeploymentKVPrefix, depId, "topology/nodes", nodeName)
	if kp, _, err := e.kv.Get(nodePath+"/type", nil); err != nil {
		return err
	} else if kp == nil {
		return fmt.Errorf("Can't retrieve node type for node %q, in deployment %q", nodeName, depId)
	} else {
		if string(kp.Value) == "janus.nodes.openstack.BlockStorage" {
			if kp, _, err = e.kv.Get(nodePath+"/properties/deletable", nil); err != nil {
				return err
			} else if kp == nil || strings.ToLower(string(kp.Value)) != "true" {
				// False by default
				log.Printf("Node %q is a BlockStorage without the property 'deletable' do not destroy it...", nodeName)
				return nil
			}
		}
	}

	infraPath := filepath.Join("work", "deployments", depId, "infra", nodeName)
	cmd := exec.CommandContext(ctx, "terraform", "destroy", "-force")
	cmd.Dir = infraPath
	errbuf := NewWriterSize(e.kv, depId)
	out := NewWriterSize(e.kv, depId)
	cmd.Stdout = out
	cmd.Stderr = errbuf

	quit := make(chan bool)
	out.run(quit)
	errbuf.run(quit)

	if err := cmd.Start(); err != nil {
		log.Print(err)
	}

	err := cmd.Wait()
	quit <- true

	return err

}
