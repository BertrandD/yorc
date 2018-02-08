package plugin

import (
	"context"
	"github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/prov"
	"testing"
	"time"
)

type mockInfraUsageCollector struct {
	getUsageInfoCalled bool
	ctx                context.Context
	conf               config.Configuration
	taskID             string
	contextCancelled   bool
}

func (m *mockInfraUsageCollector) GetUsageInfo(ctx context.Context, conf config.Configuration, taskID string) (map[string]string, error) {
	m.getUsageInfoCalled = true
	m.ctx = ctx
	m.conf = conf
	m.taskID = taskID

	go func() {
		<-m.ctx.Done()
		m.contextCancelled = true
	}()
	if m.taskID == "TestCancel" {
		<-m.ctx.Done()
	}
	if m.taskID == "TestFailure" {
		return nil, NewRPCError(errors.New("a failure occurred during plugin infra usage collector"))
	}
	res := make(map[string]string)
	res["keyOne"] = "valueOne"
	res["keyTwo"] = "valueTwo"
	res["keyThree"] = "valueThree"
	return res, nil
}

func TestInfraUsageCollectorGetUsageInfo(t *testing.T) {
	t.Parallel()
	mock := new(mockInfraUsageCollector)
	client, _ := plugin.TestPluginRPCConn(t, map[string]plugin.Plugin{
		InfraUsageCollectorPluginName: &InfraUsageCollectorPlugin{F: func() prov.InfraUsageCollector {
			return mock
		}},
	})
	defer client.Close()

	raw, err := client.Dispense(InfraUsageCollectorPluginName)
	require.Nil(t, err)

	plugin := raw.(prov.InfraUsageCollector)

	info, err := plugin.GetUsageInfo(context.Background(), config.Configuration{ConsulAddress: "test", ConsulDatacenter: "testdc"}, "TestTaskID")
	require.Nil(t, err)
	require.True(t, mock.getUsageInfoCalled)
	require.Equal(t, "test", mock.conf.ConsulAddress)
	require.Equal(t, "testdc", mock.conf.ConsulDatacenter)
	require.Equal(t, "TestTaskID", mock.taskID)
	require.Equal(t, 3, len(info))

	val, exist := info["keyOne"]
	require.True(t, exist)
	require.Equal(t, "valueOne", val)

	val, exist = info["keyTwo"]
	require.True(t, exist)
	require.Equal(t, "valueTwo", val)

	val, exist = info["keyThree"]
	require.True(t, exist)
	require.Equal(t, "valueThree", val)
}

func TestInfraUsageCollectorGetUsageInfoWithFailure(t *testing.T) {
	t.Parallel()
	mock := new(mockInfraUsageCollector)
	client, _ := plugin.TestPluginRPCConn(t, map[string]plugin.Plugin{
		InfraUsageCollectorPluginName: &InfraUsageCollectorPlugin{F: func() prov.InfraUsageCollector {
			return mock
		}},
	})
	defer client.Close()

	raw, err := client.Dispense(InfraUsageCollectorPluginName)
	require.Nil(t, err)

	plugin := raw.(prov.InfraUsageCollector)

	_, err = plugin.GetUsageInfo(context.Background(), config.Configuration{ConsulAddress: "test", ConsulDatacenter: "testdc"}, "TestFailure")
	require.Error(t, err, "An error was expected during executing plugin infra usage collector")
}

func TestInfraUsageCollectorGetUsageInfoWithCancel(t *testing.T) {
	t.Parallel()
	mock := new(mockInfraUsageCollector)
	client, _ := plugin.TestPluginRPCConn(t, map[string]plugin.Plugin{
		InfraUsageCollectorPluginName: &InfraUsageCollectorPlugin{F: func() prov.InfraUsageCollector {
			return mock
		}},
	})
	defer client.Close()

	raw, err := client.Dispense(InfraUsageCollectorPluginName)
	require.Nil(t, err)

	plugin := raw.(prov.InfraUsageCollector)

	ctx := context.Background()
	ctx, cancelF := context.WithCancel(ctx)
	go func() {
		_, err = plugin.GetUsageInfo(ctx, config.Configuration{ConsulAddress: "test", ConsulDatacenter: "testdc"}, "TestCancel")
		require.Nil(t, err)
	}()
	cancelF()
	// Wait for cancellation signal to be dispatched
	time.Sleep(50 * time.Millisecond)
	require.True(t, mock.contextCancelled, "Context should be cancelled")
}

func TestGetSupportedInfra(t *testing.T) {
	t.Parallel()
	mock := new(mockInfraUsageCollector)
	client, _ := plugin.TestPluginRPCConn(t, map[string]plugin.Plugin{
		InfraUsageCollectorPluginName: &InfraUsageCollectorPlugin{
			F: func() prov.InfraUsageCollector {
				return mock
			},
			SupportedInfra: "myInfra",
		},
	})
	defer client.Close()

	raw, err := client.Dispense(InfraUsageCollectorPluginName)
	require.Nil(t, err)

	plugin := raw.(InfraUsageCollector)

	infra, err := plugin.GetSupportedInfra()
	require.Nil(t, err)
	require.Equal(t, "myInfra", infra)
}
