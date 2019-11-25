package registrationservice

import (
	"fmt"
	"github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/host-operator/pkg/apis"
	"github.com/codeready-toolchain/toolchain-common/pkg/template"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"testing"
)

func TestDeploymentAsset(t *testing.T) {
	// given
	s := scheme.Scheme
	err := apis.AddToScheme(s)
	require.NoError(t, err)
	deploymentTemplate, err := getDeploymentTemplate(s)
	require.NoError(t, err)
	vars := getVars(&v1alpha1.RegistrationService{
		ObjectMeta: v1.ObjectMeta{Namespace: "matousjobanek-toolchain-host-operator"},
		Spec: v1alpha1.RegistrationServiceSpec{
			Image:       "quay.io/matousjobanek/registration-service:1574775161",
			Environment: "prod",
		},
	})
	cl := test.NewFakeClient(t)
	processor := template.NewProcessor(cl, s)

	// when
	objects, err := processor.Process(deploymentTemplate, vars)

	// then
	require.NoError(t, err)
	for _, object := range objects {
		fmt.Println(fmt.Sprintf("%+v", object.Object))
	}
}
