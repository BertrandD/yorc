package commands

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test the following args:
// ./janus server --infrastructure_infra1_auth_url http://localhost:5000/v2.0 --infrastructure_infra1_tenant_name validation
func TestServerInitInfraExtraFlagsWithSpaceDelimiter(t *testing.T) {
	args := []string{"./janus server", "--infrastructure_infra1_auth_url", "http://localhost:5000/v2.0", "--infrastructure_infra1_tenant_name", "validation"}
	serverInitExtraFlags(args)

	require.Len(t, args, 5)
	require.Len(t, resolvedServerExtraParams[0].viperNames, 2)
	require.Equal(t, resolvedServerExtraParams[0].viperNames[0], "infrastructures.infra1.auth_url")
	require.Equal(t, resolvedServerExtraParams[0].viperNames[1], "infrastructures.infra1.tenant_name")
}

// Test the following args:
// ./janus server --infrastructure_infra2_private_network_name=mag3-janus-network --infrastructure_infra2_region=regionOne
func TestServerInitInfraExtraFlagsWithEqualDelimiter(t *testing.T) {
	args := []string{"./janus server", "--infrastructure_infra2_private_network_name=mag3-janus-network", "--infrastructure_infra2_region=regionOne"}
	serverInitExtraFlags(args)
	require.Len(t, args, 3)
	require.Len(t, resolvedServerExtraParams[0].viperNames, 2)
	require.Equal(t, resolvedServerExtraParams[0].viperNames[0], "infrastructures.infra2.private_network_name")
	require.Equal(t, resolvedServerExtraParams[0].viperNames[1], "infrastructures.infra2.region")
}

// Test the following args:
// ./janus server --infrastructure_infra3_private_network_name=mag3-janus-network --infrastructure_infra3_region regionOne
func TestServerInitInfraExtraFlagsWithSpaceAndEqualDelimiters(t *testing.T) {
	args := []string{"./janus server", "--infrastructure_infra3_private_network_name=mag3-janus-network", "--infrastructure_infra3_region", "regionOne"}
	serverInitExtraFlags(args)

	require.Len(t, args, 4)
	require.Len(t, resolvedServerExtraParams[0].viperNames, 2)
	require.Equal(t, resolvedServerExtraParams[0].viperNames[0], "infrastructures.infra3.private_network_name")
	require.Equal(t, resolvedServerExtraParams[0].viperNames[1], "infrastructures.infra3.region")
}

// Test the following args:
// ./janus server --infrastructure_infra4_auth_url http://localhost:5000/v2.0 --infrastructure_infra4_secured --infrastructure_infra4_tenant_name validation
func TestServerInitInfraExtraFlagsWithSpaceDelimiterAndBool(t *testing.T) {
	args := []string{"./janus server", "--infrastructure_infra4_auth_url", "http://localhost:5000/v2.0", "--infrastructure_infra4_secured", "--infrastructure_infra4_tenant_name", "validation"}
	serverInitExtraFlags(args)

	require.Len(t, args, 6)
	require.Len(t, resolvedServerExtraParams[0].viperNames, 3)
	require.Equal(t, resolvedServerExtraParams[0].viperNames[0], "infrastructures.infra4.auth_url")
	require.Equal(t, resolvedServerExtraParams[0].viperNames[1], "infrastructures.infra4.secured")
	require.Equal(t, resolvedServerExtraParams[0].viperNames[2], "infrastructures.infra4.tenant_name")
}

// Test the following args:
// ./janus server --infrastructure_infra4_auth_url http://localhost:5000/v2.0 --infrastructure_infra4_tenant_name validation --infrastructure_infra4_secured
func TestServerInitInfraExtraFlagsWithSpaceDelimiterAndBoolAtEnd(t *testing.T) {
	args := []string{"./janus server", "--infrastructure_infra5_auth_url", "http://localhost:5000/v2.0", "--infrastructure_infra5_tenant_name", "validation", "--infrastructure_infra5_secured"}
	serverInitExtraFlags(args)

	require.Len(t, args, 6)
	require.Len(t, resolvedServerExtraParams[0].viperNames, 3)
	require.Equal(t, resolvedServerExtraParams[0].viperNames[0], "infrastructures.infra5.auth_url")
	require.Equal(t, resolvedServerExtraParams[0].viperNames[1], "infrastructures.infra5.tenant_name")
	require.Equal(t, resolvedServerExtraParams[0].viperNames[2], "infrastructures.infra5.secured")
}

// Test the following args:
// ./janus server --vault_auth_url http://localhost:5000/v2.0 --vault_tenant_name validation
func TestServerInitVaultExtraFlagsWithSpaceDelimiter(t *testing.T) {
	args := []string{"./janus server", "--vault_auth_url", "http://localhost:5000/v2.0", "--vault_tenant_name", "validation"}
	serverInitExtraFlags(args)

	require.Len(t, args, 5)
	require.Len(t, resolvedServerExtraParams[1].viperNames, 2)
	require.Equal(t, resolvedServerExtraParams[1].viperNames[0], "vault.auth_url")
	require.Equal(t, resolvedServerExtraParams[1].viperNames[1], "vault.tenant_name")
}

// Test the following args:
// ./janus server --vault_private_network_name=mag3-janus-network --vault_region=regionOne
func TestServerInitVaultExtraFlagsWithEqualDelimiter(t *testing.T) {
	args := []string{"./janus server", "--vault_private_network_name=mag3-janus-network", "--vault_region=regionOne"}
	serverInitExtraFlags(args)
	require.Len(t, args, 3)
	require.Len(t, resolvedServerExtraParams[1].viperNames, 2)
	require.Equal(t, resolvedServerExtraParams[1].viperNames[0], "vault.private_network_name")
	require.Equal(t, resolvedServerExtraParams[1].viperNames[1], "vault.region")
}

// Test the following args:
// ./janus server --vault_public_network_name=mag3-janus-network --vault_region2 regionOne
func TestServerInitVaultExtraFlagsWithSpaceAndEqualDelimiters(t *testing.T) {
	args := []string{"./janus server", "--vault_public_network_name=mag3-janus-network", "--vault_region2", "regionOne"}
	serverInitExtraFlags(args)

	require.Len(t, args, 4)
	require.Len(t, resolvedServerExtraParams[1].viperNames, 2)
	require.Equal(t, resolvedServerExtraParams[1].viperNames[0], "vault.public_network_name")
	require.Equal(t, resolvedServerExtraParams[1].viperNames[1], "vault.region2")
}

// Test the following args:
// ./janus server --vault_auth_url2 http://localhost:5000/v2.0 --vault_secured2 --vault_tenant_name2 validation
func TestServerInitVaultExtraFlagsWithSpaceDelimiterAndBool(t *testing.T) {
	args := []string{"./janus server", "--vault_auth_url2", "http://localhost:5000/v2.0", "--vault_secured2", "--vault_tenant_name2", "validation"}
	serverInitExtraFlags(args)

	require.Len(t, args, 6)
	require.Len(t, resolvedServerExtraParams[1].viperNames, 3)
	require.Equal(t, resolvedServerExtraParams[1].viperNames[0], "vault.auth_url2")
	require.Equal(t, resolvedServerExtraParams[1].viperNames[1], "vault.secured2")
	require.Equal(t, resolvedServerExtraParams[1].viperNames[2], "vault.tenant_name2")
}

// Test the following args:
// ./janus server --vault_auth_url3 http://localhost:5000/v2.0 --vault_tenant_name3 validation --vault_secured
func TestServerInitVaultExtraFlagsWithSpaceDelimiterAndBoolAtEnd(t *testing.T) {
	args := []string{"./janus server", "--vault_auth_url3", "http://localhost:5000/v2.0", "--vault_tenant_name3", "validation", "--vault_secured3"}
	serverInitExtraFlags(args)

	require.Len(t, args, 6)
	require.Len(t, resolvedServerExtraParams[1].viperNames, 3)
	require.Equal(t, resolvedServerExtraParams[1].viperNames[0], "vault.auth_url3")
	require.Equal(t, resolvedServerExtraParams[1].viperNames[1], "vault.tenant_name3")
	require.Equal(t, resolvedServerExtraParams[1].viperNames[2], "vault.secured3")
}
