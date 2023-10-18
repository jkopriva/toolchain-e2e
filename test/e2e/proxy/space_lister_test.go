package proxy

import (
	"context"
	"fmt"
	"testing"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	commonproxy "github.com/codeready-toolchain/toolchain-common/pkg/proxy"
	testspace "github.com/codeready-toolchain/toolchain-common/pkg/test/space"
	. "github.com/codeready-toolchain/toolchain-e2e/testsupport"
	"github.com/codeready-toolchain/toolchain-e2e/testsupport/cleanup"
	. "github.com/codeready-toolchain/toolchain-e2e/testsupport/proxy"
	proxysupport "github.com/codeready-toolchain/toolchain-e2e/testsupport/proxy"
	tsspace "github.com/codeready-toolchain/toolchain-e2e/testsupport/space"
	"github.com/codeready-toolchain/toolchain-e2e/testsupport/wait"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpaceLister(t *testing.T) {
	t.Parallel()
	// given
	awaitilities := WaitForDeployments(t)
	hostAwait := awaitilities.Host()
	memberAwait := awaitilities.Member1()
	memberAwait2 := awaitilities.Member2()

	createAndWaitForSpace(t, "appst-space", "appstudio", memberAwait)

	//SetAppstudioConfig(t, hostAwait, memberAwait)

	t.Logf("Proxy URL: %s", hostAwait.APIProxyURL)

	// carUsername := GetGeneratedName("car")
	// busUsername := GetGeneratedName("bus")
	// bicycleUsername := GetGeneratedName("road.bicycle")

	// users := map[string]*ProxyUser{
	// 	carUsername: {
	// 		ExpectedMemberCluster: memberAwait,
	// 		Username:              carUsername,
	// 		IdentityID:            uuid.Must(uuid.NewV4()),
	// 	},
	// 	busUsername: {
	// 		ExpectedMemberCluster: memberAwait2,
	// 		Username:              busUsername,
	// 		IdentityID:            uuid.Must(uuid.NewV4()),
	// 	},
	// 	bicycleUsername: {
	// 		ExpectedMemberCluster: memberAwait,
	// 		Username:              bicycleUsername, // contains a '.' that is valid in the Username but should not be in the impersonation header since it should use the compliant Username
	// 		IdentityID:            uuid.Must(uuid.NewV4()),
	// 	},
	// }
	appStudioTierRolesWSOption := commonproxy.WithAvailableRoles([]string{"admin", "contributor", "maintainer"})

	// // create the users before the subtests, so they exist for the duration of the whole test
	// for _, user := range users {
	// 	CreateAppStudioUser(t, awaitilities, user)
	// }
	// users[carUsername].ShareSpaceWith(t, hostAwait, users[busUsername])
	// users[carUsername].ShareSpaceWith(t, hostAwait, users[bicycleUsername])
	// users[busUsername].ShareSpaceWith(t, hostAwait, users[bicycleUsername])

	carUser := CreateProxyUser(t, awaitilities, "car", memberAwait)
	busUser := CreateProxyUser(t, awaitilities, "bus", memberAwait)
	bicycleUser := CreateProxyUser(t, awaitilities, "road.bicycle", memberAwait2)

	carUser.ShareSpaceWith(t, hostAwait, busUser)
	carUser.ShareSpaceWith(t, hostAwait, bicycleUser)
	busUser.ShareSpaceWith(t, hostAwait, bicycleUser)

	//usersToBeTested := []*ProxyUser{carUser, busUser, bicycleUser}
	//actionsToBeTested := []func(*testing.T, *ProxyUser, wait.Awaitilities){validateListWorkspaces}
	//[]func(*testing.T, ProxyUser, wait.Awaitilities)

	//validate(t, awaitilities, usersToBeTested, actionsToBeTested)

	t.Run("car lists workspaces", func(t *testing.T) {
		// when
		workspaces := carUser.ListWorkspaces(t, hostAwait)

		// then
		// car should see only car's workspace
		require.Len(t, workspaces, 1)
		verifyHasExpectedWorkspace(t, expectedWorkspaceFor(t, awaitilities.Host(), carUser, commonproxy.WithType("home")), workspaces...)
	})

	t.Run("car gets workspaces", func(t *testing.T) {
		t.Run("can get car workspace", func(t *testing.T) {
			// when
			workspace, err := carUser.GetWorkspace(t, hostAwait, carUser.CompliantUsername)

			// then
			require.NoError(t, err)
			verifyHasExpectedWorkspace(t, expectedWorkspaceFor(t, awaitilities.Host(), carUser, commonproxy.WithType("home"), appStudioTierRolesWSOption), *workspace)
		})

		t.Run("cannot get bus workspace", func(t *testing.T) {
			// when
			workspace, err := carUser.GetWorkspace(t, hostAwait, busUser.CompliantUsername)

			// then
			require.EqualError(t, err, "the server could not find the requested resource (get workspaces.toolchain.dev.openshift.com bus)")
			assert.Empty(t, workspace)
		})
	})

	t.Run("bus lists workspaces", func(t *testing.T) {
		// when
		workspaces := busUser.ListWorkspaces(t, hostAwait)

		// then
		// bus should see both its own and car's workspace
		require.Len(t, workspaces, 2)
		verifyHasExpectedWorkspace(t, expectedWorkspaceFor(t, awaitilities.Host(), busUser, commonproxy.WithType("home")), workspaces...)
		verifyHasExpectedWorkspace(t, expectedWorkspaceFor(t, awaitilities.Host(), carUser), workspaces...)
	})

	t.Run("bus gets workspaces", func(t *testing.T) {
		t.Run("can get bus workspace", func(t *testing.T) {
			// when
			busWS, err := busUser.GetWorkspace(t, hostAwait, busUser.CompliantUsername)

			// then
			require.NoError(t, err)
			verifyHasExpectedWorkspace(t, expectedWorkspaceFor(t, awaitilities.Host(), busUser, commonproxy.WithType("home"), appStudioTierRolesWSOption), *busWS)
		})

		t.Run("can get car workspace(bus)", func(t *testing.T) {
			// when
			carWS, err := busUser.GetWorkspace(t, hostAwait, carUser.CompliantUsername)

			// then
			require.NoError(t, err)
			verifyHasExpectedWorkspace(t, expectedWorkspaceFor(t, awaitilities.Host(), carUser, appStudioTierRolesWSOption), *carWS)
		})
	})

	t.Run("bicycle lists workspaces", func(t *testing.T) {
		// when
		workspaces := bicycleUser.ListWorkspaces(t, hostAwait)

		// then
		// car should see only car's workspace
		require.Len(t, workspaces, 3)
		verifyHasExpectedWorkspace(t, expectedWorkspaceFor(t, awaitilities.Host(), bicycleUser, commonproxy.WithType("home")), workspaces...)
		verifyHasExpectedWorkspace(t, expectedWorkspaceFor(t, awaitilities.Host(), carUser), workspaces...)
		verifyHasExpectedWorkspace(t, expectedWorkspaceFor(t, awaitilities.Host(), busUser), workspaces...)
	})

	t.Run("bicycle gets workspaces", func(t *testing.T) {
		t.Run("can get bus workspace", func(t *testing.T) {
			// when
			busWS, err := bicycleUser.GetWorkspace(t, hostAwait, busUser.CompliantUsername)

			// then
			require.NoError(t, err)
			verifyHasExpectedWorkspace(t, expectedWorkspaceFor(t, awaitilities.Host(), busUser, appStudioTierRolesWSOption), *busWS)
		})

		t.Run("can get car workspace(bicycle)", func(t *testing.T) {
			// when
			carWS, err := bicycleUser.GetWorkspace(t, hostAwait, carUser.CompliantUsername)

			// then
			require.NoError(t, err)
			verifyHasExpectedWorkspace(t, expectedWorkspaceFor(t, awaitilities.Host(), carUser, appStudioTierRolesWSOption), *carWS)
		})

		t.Run("can get bicycle workspace", func(t *testing.T) {
			// when
			bicycleWS, err := bicycleUser.GetWorkspace(t, hostAwait, bicycleUser.CompliantUsername)

			// then
			require.NoError(t, err)
			verifyHasExpectedWorkspace(t, expectedWorkspaceFor(t, awaitilities.Host(), bicycleUser, commonproxy.WithType("home"), appStudioTierRolesWSOption), *bicycleWS)
		})
	})

	t.Run("other workspace actions not permitted", func(t *testing.T) {
		t.Run("create not allowed", func(t *testing.T) {
			// given
			workspaceToCreate := expectedWorkspaceFor(t, awaitilities.Host(), busUser)
			bicycleCl, err := hostAwait.CreateAPIProxyClient(t, bicycleUser.Token, hostAwait.APIProxyURL)
			require.NoError(t, err)

			// when
			// bicycle user tries to create a workspace
			err = bicycleCl.Create(context.TODO(), &workspaceToCreate)

			// then
			require.EqualError(t, err, fmt.Sprintf("workspaces.toolchain.dev.openshift.com is forbidden: User \"%s\" cannot create resource \"workspaces\" in API group \"toolchain.dev.openshift.com\" at the cluster scope", bicycleUser.CompliantUsername))
		})

		t.Run("delete not allowed", func(t *testing.T) {
			// given
			workspaceToDelete, err := bicycleUser.GetWorkspace(t, hostAwait, bicycleUser.CompliantUsername)
			require.NoError(t, err)
			bicycleCl, err := hostAwait.CreateAPIProxyClient(t, bicycleUser.Token, hostAwait.APIProxyURL)
			require.NoError(t, err)

			// bicycle user tries to delete a workspace
			err = bicycleCl.Delete(context.TODO(), workspaceToDelete)

			// then
			require.EqualError(t, err, fmt.Sprintf("workspaces.toolchain.dev.openshift.com \"%[1]s\" is forbidden: User \"%[1]s\" cannot delete resource \"workspaces\" in API group \"toolchain.dev.openshift.com\" at the cluster scope", bicycleUser.CompliantUsername))
		})

		t.Run("update not allowed", func(t *testing.T) {
			// when
			workspaceToUpdate := expectedWorkspaceFor(t, awaitilities.Host(), bicycleUser, commonproxy.WithType("home"))
			bicycleCl, err := hostAwait.CreateAPIProxyClient(t, bicycleUser.Token, hostAwait.APIProxyURL)
			require.NoError(t, err)

			// bicycle user tries to update a workspace
			err = bicycleCl.Update(context.TODO(), &workspaceToUpdate)

			// then
			require.EqualError(t, err, fmt.Sprintf("workspaces.toolchain.dev.openshift.com \"%[1]s\" is forbidden: User \"%[1]s\" cannot update resource \"workspaces\" in API group \"toolchain.dev.openshift.com\" at the cluster scope", bicycleUser.CompliantUsername))
		})
	})
}

func createAndWaitForSpace(t *testing.T, name, tierName string, targetCluster *wait.MemberAwaitility) {
	awaitilities := WaitForDeployments(t)
	hostAwait := awaitilities.Host()
	space := testspace.NewSpace(hostAwait.Namespace, name, testspace.WithTierName(tierName), testspace.WithSpecTargetCluster(targetCluster.ClusterName))
	err := hostAwait.Client.Create(context.TODO(), space)
	require.NoError(t, err)

	_, _, binding := tsspace.CreateMurWithAdminSpaceBindingForSpace(t, awaitilities, space, true)

	tier, err := hostAwait.WaitForNSTemplateTier(t, tierName)
	require.NoError(t, err)

	_, err = targetCluster.WaitForNSTmplSet(t, space.Name,
		wait.UntilNSTemplateSetHasConditions(wait.Provisioned()),
		wait.UntilNSTemplateSetHasSpaceRoles(
			wait.SpaceRole(tier.Spec.SpaceRoles[binding.Spec.SpaceRole].TemplateRef, binding.Spec.MasterUserRecord)))
	require.NoError(t, err)

	_, err = hostAwait.WaitForSpace(t, space.Name,
		wait.UntilSpaceHasConditions(wait.Provisioned()))
	require.NoError(t, err)
	cleanup.AddCleanTasks(t, awaitilities.Host().Client, space)
}

func expectedWorkspaceFor(t *testing.T, hostAwait *wait.HostAwaitility, user *proxysupport.ProxyUser, additionalWSOptions ...commonproxy.WorkspaceOption) toolchainv1alpha1.Workspace {
	space, err := hostAwait.WaitForSpace(t, user.CompliantUsername, wait.UntilSpaceHasAnyTargetClusterSet(), wait.UntilSpaceHasAnyTierNameSet())
	require.NoError(t, err)

	commonWSoptions := []commonproxy.WorkspaceOption{
		commonproxy.WithObjectMetaFrom(space.ObjectMeta),
		commonproxy.WithNamespaces([]toolchainv1alpha1.SpaceNamespace{
			{
				Name: user.CompliantUsername + "-tenant",
				Type: "default",
			},
		}),
		commonproxy.WithOwner(user.Signup.Name),
		commonproxy.WithRole("admin"),
	}
	ws := commonproxy.NewWorkspace(user.CompliantUsername,
		append(commonWSoptions, additionalWSOptions...)...,
	)
	return *ws
}

func verifyHasExpectedWorkspace(t *testing.T, expectedWorkspace toolchainv1alpha1.Workspace, actualWorkspaces ...toolchainv1alpha1.Workspace) {
	for _, actualWorkspace := range actualWorkspaces {
		if actualWorkspace.Name == expectedWorkspace.Name {
			assert.Equal(t, expectedWorkspace.Status, actualWorkspace.Status)
			assert.NotEmpty(t, actualWorkspace.ObjectMeta.ResourceVersion, "Workspace.ObjectMeta.ResourceVersion field is empty: %#v", actualWorkspace)
			assert.NotEmpty(t, actualWorkspace.ObjectMeta.Generation, "Workspace.ObjectMeta.Generation field is empty: %#v", actualWorkspace)
			assert.NotEmpty(t, actualWorkspace.ObjectMeta.CreationTimestamp, "Workspace.ObjectMeta.CreationTimestamp field is empty: %#v", actualWorkspace)
			assert.NotEmpty(t, actualWorkspace.ObjectMeta.UID, "Workspace.ObjectMeta.UID field is empty: %#v", actualWorkspace)
			return
		}
	}
	t.Errorf("expected workspace %s not found", expectedWorkspace.Name)
}
