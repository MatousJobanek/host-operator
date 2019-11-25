package registrationservice

import (
	"context"
	"fmt"
	errs "github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"
	"github.com/codeready-toolchain/host-operator/pkg/configuration"
	"github.com/codeready-toolchain/toolchain-common/pkg/template"
	"github.com/openshift/api/template/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_registrationservice")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new RegistrationService Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	deploymentTemplate, err := getDeploymentTemplate(mgr.GetScheme())
	if err != nil {
		return errs.Wrap(err, "unable to decode the registration service deployment")
	}

	configuration.CreateEmptyRegistry()
	return add(mgr, newReconciler(mgr, deploymentTemplate))
}

func getDeploymentTemplate(s *runtime.Scheme) (*v1.Template, error) {
	deployment, err := Asset("deployment.yaml")
	if err != nil {
		return nil, err
	}
	decoder := serializer.NewCodecFactory(s).UniversalDeserializer()
	deploymentTemplate := &v1.Template{}
	_, _, err = decoder.Decode([]byte(deployment), nil, deploymentTemplate)
	return deploymentTemplate, err
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, regServiceDeployment *v1.Template) reconcile.Reconciler {
	return &ReconcileRegistrationService{
		client:               mgr.GetClient(),
		scheme:               mgr.GetScheme(),
		regServiceDeployment: regServiceDeployment,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("registrationservice-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource RegistrationService
	err = c.Watch(&source.Kind{Type: &toolchainv1alpha1.RegistrationService{}}, &handler.EnqueueRequestForObject{}, predicate.GenerationChangedPredicate{})
	if err != nil {
		return err
	}

	service := r.(*ReconcileRegistrationService)

	processor := template.NewProcessor(service.client, service.scheme)

	objects, err := processor.Process(service.regServiceDeployment.DeepCopy(), map[string]string{})
	if err != nil {
		return err
	}

	for _, object := range objects {
		if object.Object == nil {
			continue
		}
		err = c.Watch(&source.Kind{Type: object.Object}, &handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &toolchainv1alpha1.RegistrationService{},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// blank assignment to verify that ReconcileRegistrationService implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileRegistrationService{}

// ReconcileRegistrationService reconciles a RegistrationService object
type ReconcileRegistrationService struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client               client.Client
	scheme               *runtime.Scheme
	regServiceDeployment *v1.Template
}

// Reconcile reads that state of the cluster for a RegistrationService object and makes changes based on the state read
// and what is in the RegistrationService.Spec
func (r *ReconcileRegistrationService) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling RegistrationService")

	// Fetch the RegistrationService regService
	regService := &toolchainv1alpha1.RegistrationService{}
	err := r.client.Get(context.TODO(), request.NamespacedName, regService)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	processor := template.NewProcessor(r.client, r.scheme)

	objects, err := processor.Process(r.regServiceDeployment.DeepCopy(), getVars(regService))
	if err != nil {
		return reconcile.Result{}, err
	}

	fmt.Println(fmt.Sprintf("============== "))
	for _, object := range objects {

		fmt.Println("creating/updating", object.Object.GetObjectKind().GroupVersionKind())

		createdOrUpdated, err := processor.ApplySingle(object.Object, false, regService)
		if createdOrUpdated || err != nil {
			return reconcile.Result{}, err
		}

	}
	fmt.Println("nothing")

	return reconcile.Result{}, nil
}

type templateVars map[string]string

func getVars(regService *toolchainv1alpha1.RegistrationService) map[string]string {
	var vars templateVars = map[string]string{}
	vars.addIfNotEmpty("NAMESPACE", regService.Namespace)
	vars.addIfNotEmpty("IMAGE", regService.Spec.Image)
	vars.addIfNotZero("REPLICAS", regService.Spec.Replicas)
	vars.addIfNotEmpty("ENVIRONMENT", regService.Spec.Environment)
	vars.addIfNotEmpty("AUTH_CLIENT_LIBRARY_URL", regService.Spec.AuthClient.LibraryUrl)
	vars.addIfNotEmpty("AUTH_CLIENT_CONFIG_RAW", regService.Spec.AuthClient.Config)
	vars.addIfNotEmpty("AUTH_CLIENT_PUBLIC_KEYS_URL", regService.Spec.AuthClient.PublicKeysUrl)
	return vars
}

func (v *templateVars) addIfNotEmpty(key, value string) {
	if value != "" {
		(*v)[key] = value
	}
}

func (v *templateVars) addIfNotZero(key string, value int) {
	if value != 0 {
		(*v)[key] = fmt.Sprintf("%d", value)
	}
}
