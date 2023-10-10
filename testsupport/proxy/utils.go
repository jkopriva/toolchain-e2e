package proxy

import (
	"fmt"
	"testing"

	"github.com/devfile/library/v2/pkg/util"

	testconfig "github.com/codeready-toolchain/toolchain-common/pkg/test/config"
	appstudiov1 "github.com/codeready-toolchain/toolchain-e2e/testsupport/appstudio/api/v1alpha1"
	"github.com/codeready-toolchain/toolchain-e2e/testsupport/wait"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PatchStringValue struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

func TenantNsName(username string) string {
	return fmt.Sprintf("%s-tenant", username)
}

func NewApplication(applicationName, namespace string) *appstudiov1.Application {
	return &appstudiov1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      applicationName,
			Namespace: namespace,
		},
		Spec: appstudiov1.ApplicationSpec{
			DisplayName: fmt.Sprintf("Proxy test for user %s", namespace),
		},
	}
}

// SetAppstudioConfig applies toolchain configuration for appstudio scenarios
func SetAppstudioConfig(t *testing.T, hostAwait *wait.HostAwaitility, memberAwait *wait.MemberAwaitility) {
	// member cluster configured to skip user creation to mimic appstudio configuration where user & identity resources are not created
	memberConfigurationWithSkipUserCreation := testconfig.ModifyMemberOperatorConfigObj(memberAwait.GetMemberOperatorConfig(t), testconfig.SkipUserCreation(true))
	// configure default space tier to appstudio
	hostAwait.UpdateToolchainConfig(t, testconfig.Tiers().DefaultUserTier("deactivate30").DefaultSpaceTier("appstudio"), testconfig.Members().Default(memberConfigurationWithSkipUserCreation.Spec))
}

func GetGeneratedName(name string) string {
	return name + "-" + util.GenerateRandomString(4)
}
