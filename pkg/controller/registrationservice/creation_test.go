package registrationservice

import (
	"context"
	"fmt"
	"testing"

	"github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/host-operator/pkg/apis"
	"github.com/codeready-toolchain/host-operator/pkg/configuration"
	"github.com/codeready-toolchain/host-operator/test"
	"github.com/codeready-toolchain/toolchain-common/pkg/template"
	. "github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCreateOrUpdateResources(t *testing.T) {
	// given
	s := scheme.Scheme
	err := apis.AddToScheme(s)
	require.NoError(t, err)
	config, err := configuration.New("")
	require.NoError(t, err)

	t.Run("create with default values", func(t *testing.T) {
		// given
		cl := NewFakeClient(t)

		// when
		err = CreateOrUpdateResources(cl, s, HostOperatorNs, config)

		// then
		require.NoError(t, err)
		test.AssertThatRegistrationService(t, "registration-service", cl).
			HasImage("").
			HasEnvironment("prod").
			HasReplicas(0).
			HasAuthConfig("").
			HasAuthLibraryUrl("").
			HasAuthPublicKeysUrl("")

	})

	t.Run("update to RegService with image and environment values set", func(t *testing.T) {
		// given
		cl := NewFakeClient(t)
		regService := &v1alpha1.RegistrationService{
			ObjectMeta: v1.ObjectMeta{
				Namespace: HostOperatorNs,
				Name:      "registration-service",
			}}
		processor := template.NewProcessor(cl, s)
		_, err := processor.ApplySingle(regService, false, nil)
		require.NoError(t, err)
		SetEnvVarsAndRestore(t,
			Env("REGISTRATION_SERVICE_IMAGE", "quay.io/rh/registration-service:v0.1"),
			Env("REGISTRATION_SERVICE_ENVIRONMENT", "test"))

		// when
		err = CreateOrUpdateResources(cl, s, HostOperatorNs, config)

		// then
		require.NoError(t, err)
		test.AssertThatRegistrationService(t, "registration-service", cl).
			HasImage("quay.io/rh/registration-service:v0.1").
			HasEnvironment("test").
			HasReplicas(0).
			HasAuthConfig("").
			HasAuthLibraryUrl("").
			HasAuthPublicKeysUrl("")
	})

	t.Run("when creation fails then should return error", func(t *testing.T) {
		// given
		cl := NewFakeClient(t)
		cl.MockCreate = func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
			return fmt.Errorf("creation failed")
		}

		// when
		err = CreateOrUpdateResources(cl, s, HostOperatorNs, config)

		// then
		require.Error(t, err)
	})
}
