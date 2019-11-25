package registrationservice

import (
	"github.com/codeready-toolchain/api/pkg/apis"
	"github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
)

func TestReconcileRegistrationService(t *testing.T) {
	// given
	s := scheme.Scheme
	err := apis.AddToScheme(s)
	require.NoError(t, err)
	template, err := getDeploymentTemplate(s)
	require.NoError(t, err)
	reqService := &v1alpha1.RegistrationService{
		ObjectMeta: v1.ObjectMeta{
			Name:      "registration-service",
			Namespace: "host-operator",
		},
		Spec: v1alpha1.RegistrationServiceSpec{
			Image:       "quay.io/codeready-toolchain/registration-service:1574865601",
			Environment: "dev",
		}}
	service := ReconcileRegistrationService{
		client:               test.NewFakeClient(t, reqService),
		scheme:               s,
		regServiceDeployment: template,
	}
	request := reconcile.Request{NamespacedName: test.NamespacedName("host-operator", "registration-service")}

	// when
	_, err = service.Reconcile(request)

	// then
	require.NoError(t, err)

	// when
	_, err = service.Reconcile(request)

	// then
	require.NoError(t, err)

	// when
	_, err = service.Reconcile(request)

	// then
	require.NoError(t, err)

	// when
	_, err = service.Reconcile(request)

	// then
	require.NoError(t, err)

	// when
	_, err = service.Reconcile(request)

	// then
	require.NoError(t, err)

}
