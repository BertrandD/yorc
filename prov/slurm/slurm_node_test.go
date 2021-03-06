// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package slurm

import (
	"context"
	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/require"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/deployments"
	"path"
	"strconv"
	"testing"
)

func loadTestYaml(t *testing.T, kv *api.KV) string {
	deploymentID := path.Base(t.Name())
	yamlName := "testdata/" + deploymentID + ".yaml"
	err := deployments.StoreDeploymentDefinition(context.Background(), kv, deploymentID, yamlName)
	require.Nil(t, err, "Failed to parse "+yamlName+" definition")
	return deploymentID
}

func testSimpleSlurmNodeAllocation(t *testing.T, kv *api.KV, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)
	g := slurmGenerator{}
	infrastructure := infrastructure{}

	err := g.generateNodeAllocation(context.Background(), kv, cfg, deploymentID, "Compute", "0", &infrastructure)
	require.Nil(t, err)

	require.Len(t, infrastructure.nodes, 1)
	require.Equal(t, "0", infrastructure.nodes[0].instanceName)
	require.Equal(t, "gpu:1", infrastructure.nodes[0].gres)
	require.Equal(t, "debug", infrastructure.nodes[0].partition)
	require.Equal(t, "2G", infrastructure.nodes[0].memory)
	require.Equal(t, "4", infrastructure.nodes[0].cpu)
	require.Equal(t, "xyz", infrastructure.nodes[0].jobName)
}

func testSimpleSlurmNodeAllocationWithoutProps(t *testing.T, kv *api.KV, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)
	g := slurmGenerator{}
	infrastructure := infrastructure{}

	err := g.generateNodeAllocation(context.Background(), kv, cfg, deploymentID, "Compute", "0", &infrastructure)
	require.Nil(t, err)

	require.Len(t, infrastructure.nodes, 1)
	require.Equal(t, "0", infrastructure.nodes[0].instanceName)
	require.Equal(t, "", infrastructure.nodes[0].gres)
	require.Equal(t, "", infrastructure.nodes[0].partition)
	require.Equal(t, "", infrastructure.nodes[0].memory)
	require.Equal(t, "", infrastructure.nodes[0].cpu)
	require.Equal(t, "simpleSlurmNodeAllocationWithoutProps", infrastructure.nodes[0].jobName)
}

func testMultipleSlurmNodeAllocation(t *testing.T, kv *api.KV, cfg config.Configuration) {
	t.Parallel()
	deploymentID := loadTestYaml(t, kv)
	g := slurmGenerator{}
	infrastructure := infrastructure{}

	nb, err := deployments.GetDefaultNbInstancesForNode(kv, deploymentID, "Compute")
	require.Nil(t, err)
	require.Equal(t, uint32(4), nb)

	for i := 0; i < int(nb); i++ {
		istr := strconv.Itoa(i)
		err := g.generateNodeAllocation(context.Background(), kv, cfg, deploymentID, "Compute", istr, &infrastructure)
		require.Nil(t, err)

		require.Len(t, infrastructure.nodes, i+1)
		require.Equal(t, istr, infrastructure.nodes[i].instanceName)
		require.Equal(t, "gpu:1", infrastructure.nodes[i].gres)
		require.Equal(t, "debug", infrastructure.nodes[i].partition)
		require.Equal(t, "2G", infrastructure.nodes[i].memory)
		require.Equal(t, "4", infrastructure.nodes[i].cpu)
		require.Equal(t, "xyz", infrastructure.nodes[i].jobName)
	}
}
