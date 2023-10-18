package parallel

import (
	"context"
	"sync"
	"testing"

	"github.com/gofrs/uuid"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/api/v1alpha1"
	testspace "github.com/codeready-toolchain/toolchain-common/pkg/test/space"
	"github.com/codeready-toolchain/toolchain-e2e/testsupport"

	"github.com/codeready-toolchain/toolchain-e2e/testsupport/cleanup"
	tsspace "github.com/codeready-toolchain/toolchain-e2e/testsupport/space"
	"github.com/codeready-toolchain/toolchain-e2e/testsupport/tiers"
	"github.com/codeready-toolchain/toolchain-e2e/testsupport/wait"

	"github.com/stretchr/testify/require"

	commonproxy "github.com/codeready-toolchain/toolchain-common/pkg/proxy"
	proxysupport "github.com/codeready-toolchain/toolchain-e2e/testsupport/proxy"

	"github.com/stretchr/testify/assert"
)

const (
	AppStudioSpace = "appstudio" //-space
	AppStudioUser  = "appstudio-user"
)

type ProxyRunner struct {
	Awaitilities wait.Awaitilities
	WithCleanup  bool
}

func (r *ProxyRunner) Run(t *testing.T) {
	var wg sync.WaitGroup

	toRun := []func(t *testing.T){
		r.prepareAppStudioProvisionedSpace,
		//r.prepareAppStudioProvisionedUsers,
	}

	for _, funcToRun := range toRun {
		wg.Add(1)
		go func(run func(t *testing.T)) {
			defer wg.Done()
			run(t)
		}(funcToRun)
	}

	wg.Wait()
}

func TestProxy(t *testing.T) {
	t.Run("Run proxy tests in parallel", func(t *testing.T) {
		t.Parallel()
		awaitilities := testsupport.WaitForDeployments(t)
		runner := ProxyRunner{
			Awaitilities: awaitilities,
			WithCleanup:  true,
		}
		runner.Run(t)
		runner.runVerifyFunctions(t, awaitilities)
	})
}

func (r *ProxyRunner) prepareAppStudioProvisionedSpace(t *testing.T) {
	r.createAndWaitForSpace(t, AppStudioSpace, "appstudio", r.Awaitilities.Member1())
}

func (r *ProxyRunner) createAndWaitForSpace(t *testing.T, name, tierName string, targetCluster *wait.MemberAwaitility) {
	hostAwait := r.Awaitilities.Host()
	space := testspace.NewSpace(r.Awaitilities.Host().Namespace, name, testspace.WithTierName(tierName), testspace.WithSpecTargetCluster(targetCluster.ClusterName))
	err := hostAwait.Client.Create(context.TODO(), space)
	require.NoError(t, err)

	_, _, binding := tsspace.CreateMurWithAdminSpaceBindingForSpace(t, r.Awaitilities, space, r.WithCleanup)

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
	if r.WithCleanup {
		cleanup.AddCleanTasks(t, r.Awaitilities.Host().Client, space)
	}
}

func (r *ProxyRunner) prepareAppStudioProvisionedUsers(t *testing.T) {
	r.prepareAppStudioProvisionedUser(t, AppStudioUser)
	//r.prepareAppStudioProvisionedUser(t, "car")
}

func (r *ProxyRunner) prepareAppStudioProvisionedUser(t *testing.T, userName string) {
	usersignup := r.prepareUser(t, userName, r.Awaitilities.Member1())
	hostAwait := r.Awaitilities.Host()

	// promote to appstudio
	tiers.MoveSpaceToTier(t, hostAwait, usersignup.Status.CompliantUsername, "appstudio")

	t.Logf("user %s was promoted to appstudio tier", userName)

	// verify that it's promoted
	_, err := r.Awaitilities.Host().WaitForMasterUserRecord(t, usersignup.Status.CompliantUsername,
		wait.UntilMasterUserRecordHasConditions(wait.Provisioned(), wait.ProvisionedNotificationCRCreated()))
	require.NoError(t, err)
}

func (r *ProxyRunner) prepareUser(t *testing.T, name string, targetCluster *wait.MemberAwaitility) *toolchainv1alpha1.UserSignup {
	requestBuilder := testsupport.NewSignupRequest(r.Awaitilities).
		Username(name).
		UserID(uuid.Must(uuid.NewV4()).String()).
		AccountID(uuid.Must(uuid.NewV4()).String()).
		OriginalSub("original_sub_" + name).
		ManuallyApprove().
		TargetCluster(targetCluster)
	if !r.WithCleanup {
		requestBuilder = requestBuilder.DisableCleanup()
	}

	signup, _ := requestBuilder.
		RequireConditions(wait.ConditionSet(wait.Default(), wait.ApprovedByAdmin())...).
		Execute(t).
		Resources()
	_, err := r.Awaitilities.Host().WaitForMasterUserRecord(t, signup.Status.CompliantUsername,
		wait.UntilMasterUserRecordHasConditions(wait.Provisioned(), wait.ProvisionedNotificationCRCreated()))
	require.NoError(t, err)
	return signup
}

func (r *ProxyRunner) runVerifyFunctions(t *testing.T, awaitilities wait.Awaitilities) {
	var wg sync.WaitGroup

	//memberAwait := awaitilities.Member1()
	//memberAwait2 := awaitilities.Member2()
	//hostAwait := awaitilities.Host()

	//carUser := proxysupport.CreateProxyUser(t, awaitilities, "car", memberAwait)
	//busUser := proxysupport.CreateProxyUser(t, awaitilities, "bus", memberAwait)
	//bicycleUser := proxysupport.CreateProxyUser(t, awaitilities, "road.bicycle", memberAwait2)

	//carUser.ShareSpaceWith(t, hostAwait, busUser)
	//carUser.ShareSpaceWith(t, hostAwait, bicycleUser)
	//busUser.ShareSpaceWith(t, hostAwait, bicycleUser)
	//usersToBeTested := []*proxysupport.ProxyUser{carUser, busUser, bicycleUser}

	actionsToBeTested := []func(*testing.T, *proxysupport.ProxyUser, wait.Awaitilities){validateListWorkspaces, validateGetWorkspaces}

	//validate(t, awaitilities, usersToBeTested, actionsToBeTested)

	usersToBeTestedNames := []string{"car2", "bus2", "bicycle2"}
	validateNames(t, awaitilities, usersToBeTestedNames, actionsToBeTested)

	wg.Wait()

	cleanup.ExecuteAllCleanTasks(t)
}

func validate(t *testing.T, awaitilities wait.Awaitilities, usersToBeTested []*proxysupport.ProxyUser, actionsToBeTested []func(*testing.T, *proxysupport.ProxyUser, wait.Awaitilities)) {
	for _, user := range usersToBeTested {
		for _, action := range actionsToBeTested {
			actionFunc := action
			go func(userFunc *proxysupport.ProxyUser) {
				actionFunc(t, userFunc, awaitilities)
			}(user)
		}
	}
}

func validateNames(t *testing.T, awaitilities wait.Awaitilities, usersToBeTested []string, actionsToBeTested []func(*testing.T, *proxysupport.ProxyUser, wait.Awaitilities)) {
	for _, userName := range usersToBeTested {
		user := proxysupport.CreateProxyUser(t, awaitilities, userName, awaitilities.Member1())
		for _, action := range actionsToBeTested {
			actionFunc := action
			go func(userFunc *proxysupport.ProxyUser) {
				actionFunc(t, userFunc, awaitilities)
			}(user)
		}
	}
}

func validateListWorkspaces(t *testing.T, user *proxysupport.ProxyUser, awaitilities wait.Awaitilities) {
	// when
	workspaces := user.ListWorkspaces(t, awaitilities.Host())

	// then
	// user should see only users's workspace
	require.Len(t, workspaces, 1)
	verifyHasExpectedWorkspace(t, expectedWorkspaceFor(t, awaitilities.Host(), user, commonproxy.WithType("home")), workspaces...)
}

func validateGetWorkspaces(t *testing.T, user *proxysupport.ProxyUser, awaitilities wait.Awaitilities) {
	// when
	userWorkspace, err := user.GetWorkspace(t, awaitilities.Host(), user.CompliantUsername)
	appStudioTierRolesWSOption := commonproxy.WithAvailableRoles([]string{"admin", "contributor", "maintainer"})

	// then
	require.NoError(t, err)
	verifyHasExpectedWorkspace(t, expectedWorkspaceFor(t, awaitilities.Host(), user, commonproxy.WithType("home"), appStudioTierRolesWSOption), *userWorkspace)
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
			//assert.Equal(t, expectedWorkspace.Status, actualWorkspace.Status)
			assert.NotEmpty(t, actualWorkspace.ObjectMeta.ResourceVersion, "Workspace.ObjectMeta.ResourceVersion field is empty: %#v", actualWorkspace)
			assert.NotEmpty(t, actualWorkspace.ObjectMeta.Generation, "Workspace.ObjectMeta.Generation field is empty: %#v", actualWorkspace)
			assert.NotEmpty(t, actualWorkspace.ObjectMeta.CreationTimestamp, "Workspace.ObjectMeta.CreationTimestamp field is empty: %#v", actualWorkspace)
			assert.NotEmpty(t, actualWorkspace.ObjectMeta.UID, "Workspace.ObjectMeta.UID field is empty: %#v", actualWorkspace)
			return
		}
	}
	t.Errorf("expected workspace %s not found", expectedWorkspace.Name)
}
