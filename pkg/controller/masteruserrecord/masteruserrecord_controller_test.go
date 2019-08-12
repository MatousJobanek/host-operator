package masteruserrecord

import (
	"context"
	"fmt"
	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/host-operator/pkg/apis"
	"github.com/codeready-toolchain/toolchain-common/pkg/cluster"
	"github.com/codeready-toolchain/toolchain-common/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierros "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/kubefed/pkg/apis/core/common"
	"sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	"testing"
)

const (
	hostClusterName   = "host-cluster"
	memberOperatorNs  = "toolchain-member-operator"
	memberClusterName = "member-cluster"
	hostOperatorNs    = "toolchain-host-operator"
)

type getMemberCluster func(cl client.Client) func(name string) (*cluster.FedCluster, bool)

func TestCreateUserAccountSuccessful(t *testing.T) {
	// given
	logf.SetLogger(logf.ZapLogger(true))
	s := apiScheme(t)
	mur := newMasterUserRecord("john")
	memberClient := fake.NewFakeClientWithScheme(s)
	hostClient := fake.NewFakeClientWithScheme(s, mur)
	cntrl := newController(hostClient, memberClient, s, newGetMemberCluster(true, v1.ConditionTrue))

	// when
	_, err := cntrl.Reconcile(newMurRequest(mur))

	// then
	require.NoError(t, err)
	ua := &toolchainv1alpha1.UserAccount{}
	err = memberClient.Get(context.TODO(), namespacedName(memberOperatorNs, "john"), ua)
	require.NoError(t, err)
	assert.EqualValues(t, mur.Spec.UserAccounts[0].Spec, ua.Spec)
	assertMurCondition(t, hostClient, toBeNotReady(provisioningReason, ""))
}

func TestCreateUserAccountFailed(t *testing.T) {
	// given
	logf.SetLogger(logf.ZapLogger(true))
	s := apiScheme(t)
	mur := newMasterUserRecord("john")
	memberClient := fake.NewFakeClientWithScheme(s)
	hostClient := fake.NewFakeClientWithScheme(s, mur)

	t.Run("when member cluster does not exist", func(t *testing.T) {
		// given
		cntrl := newController(hostClient, memberClient, s, newGetMemberCluster(false, v1.ConditionTrue))

		// when
		_, err := cntrl.Reconcile(newMurRequest(mur))

		// then
		require.Error(t, err)
		msg := "the fedCluster member-cluster not found in the registry"
		assert.Contains(t, err.Error(), msg)
		assertUaNotFound(t, memberClient)
		assertMurCondition(t, hostClient, toBeNotReady(targetClusterNotReady, msg))
	})

	t.Run("when member cluster is not ready", func(t *testing.T) {
		// given
		cntrl := newController(hostClient, memberClient, s, newGetMemberCluster(true, v1.ConditionFalse))

		// when
		_, err := cntrl.Reconcile(newMurRequest(mur))

		// then
		require.Error(t, err)
		msg := "the fedCluster member-cluster is not ready"
		assert.Contains(t, err.Error(), msg)
		assertUaNotFound(t, memberClient)
		assertMurCondition(t, hostClient, toBeNotReady(targetClusterNotReady, msg))
	})

	t.Run("status update failed", func(t *testing.T) {
		// given
		cntrl := newController(hostClient, memberClient, s, newGetMemberCluster(true, v1.ConditionTrue))
		statusUpdater := func(mur *toolchainv1alpha1.MasterUserRecord, message string) error {
			return fmt.Errorf("unable to update status")
		}

		// when
		err := cntrl.wrapErrorWithStatusUpdate(log, mur, statusUpdater,
			apierros.NewBadRequest("oopsy woopsy"), "failed to create %s", "user bob")

		// then
		require.Error(t, err)
		assert.Equal(t, "failed to create user bob: oopsy woopsy", err.Error())
	})
}

func assertMurCondition(t *testing.T, hostClient client.Client, expected toolchainv1alpha1.Condition) {
	record := &toolchainv1alpha1.MasterUserRecord{}
	err := hostClient.Get(context.TODO(), namespacedName(hostOperatorNs, "john"), record)
	require.NoError(t, err)
	test.AssertConditionsMatch(t, record.Status.Conditions, expected)
}

func assertUaNotFound(t *testing.T, memberClient client.Client) {
	ua := &toolchainv1alpha1.UserAccount{}
	err := memberClient.Get(context.TODO(), namespacedName(memberOperatorNs, "john"), ua)
	require.Error(t, err)
	assert.IsType(t, metav1.StatusReasonNotFound, errors.ReasonForError(err))
}

func newMasterUserRecord(userName string) *toolchainv1alpha1.MasterUserRecord {
	mur := &toolchainv1alpha1.MasterUserRecord{
		ObjectMeta: metav1.ObjectMeta{
			Name:      userName,
			Namespace: hostOperatorNs,
		},
		Spec: toolchainv1alpha1.MasterUserRecordSpec{
			UserID: "12345abcdef",
			UserAccounts: []toolchainv1alpha1.UserAccountEmbedded{{
				TargetCluster: memberClusterName,
				SyncIndex:     "0",
				Spec: toolchainv1alpha1.UserAccountSpec{
					UserID:  "12345abcdef",
					NSLimit: "basic",
					NSTemplateSet: toolchainv1alpha1.NSTemplateSetSpec{
						TierName: "basic",
						Namespaces: []toolchainv1alpha1.Namespace{
							{
								Type:     "ide",
								Revision: "123abc",
								Template: "",
							},
							{
								Type:     "ci/cd",
								Revision: "123abc",
								Template: "",
							},
							{
								Type:     "staging",
								Revision: "123abc",
								Template: "",
							},
						},
					},
				},
			}},
		},
	}
	return mur
}

func newMurRequest(mur *toolchainv1alpha1.MasterUserRecord) reconcile.Request {
	return reconcile.Request{
		NamespacedName: namespacedName(mur.ObjectMeta.Namespace, mur.ObjectMeta.Name),
	}
}

func apiScheme(t *testing.T) *runtime.Scheme {
	s := scheme.Scheme
	err := apis.AddToScheme(s)
	require.NoError(t, err)
	return s
}

func newController(hostClient, memberClient client.Client, s *runtime.Scheme,
	getMemberCluster getMemberCluster) ReconcileMasterUserRecord {

	return ReconcileMasterUserRecord{
		client:                hostClient,
		scheme:                s,
		retrieveMemberCluster: getMemberCluster(memberClient),
	}

}

func newGetMemberCluster(ok bool, status v1.ConditionStatus) getMemberCluster {
	if !ok {
		return func(cl client.Client) func(name string) (*cluster.FedCluster, bool) {
			return func(name string) (*cluster.FedCluster, bool) {
				return nil, false
			}
		}
	}
	return func(cl client.Client) func(name string) (*cluster.FedCluster, bool) {
		return func(name string) (*cluster.FedCluster, bool) {
			if name != memberClusterName {
				return nil, false
			}
			return &cluster.FedCluster{
				Client:            cl,
				Type:              cluster.Host,
				OperatorNamespace: memberOperatorNs,
				OwnerClusterName:  hostClusterName,
				ClusterStatus: &v1beta1.KubeFedClusterStatus{
					Conditions: []v1beta1.ClusterCondition{{
						Type:   common.ClusterReady,
						Status: status,
					}},
				},
			}, true
		}
	}
}