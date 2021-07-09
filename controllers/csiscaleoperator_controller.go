/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	csiv1 "github.com/IBM/ibm-spectrum-scale-csi/api/v1"
	oconfig "github.com/IBM/ibm-spectrum-scale-csi/controllers/config"
	csiscaleoperator "github.com/IBM/ibm-spectrum-scale-csi/controllers/internal/csiscaleoperator"
	clustersyncer "github.com/IBM/ibm-spectrum-scale-csi/controllers/syncer"
	oversion "github.com/IBM/ibm-spectrum-scale-csi/controllers/version"
	"github.com/presslabs/controller-util/syncer"
)

// CSIScaleOperatorReconciler reconciles a CSIScaleOperator object
type CSIScaleOperatorReconciler struct {
	Client        client.Client
	Scheme        *runtime.Scheme
	recorder      record.EventRecorder
	serverVersion string
}

var daemonSetRestartedKey = ""
var daemonSetRestartedValue = ""

var csiLog = log.Log.WithName("csiscaleoperator_controller")

type reconciler func(instance *csiscaleoperator.CSIScaleOperator) error

//+kubebuilder:rbac:groups=csi.ibm.com,resources=csiscaleoperators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=csi.ibm.com,resources=csiscaleoperators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=csi.ibm.com,resources=csiscaleoperators/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CSIScaleOperator object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
/*func (r *CSIScaleOperatorReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	//_ = log.FromContext(ctx)

	logger := csiLog.WithName("Reconcile")
	logger.Info("in Reconcile(): ", req)
	// your logic here
//	instance := &csiv1.CSIScaleOperator{}
	instance := csiscaleoperator.New(&csiv1.CSIScaleOperator{})
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)
//        memcached := &cachev1alpha1.Memcached{}
 //       err := r.Get(ctx, req.NamespacedName, memcached)

	//logger.Info("req", req.NamespacedName)
	//logger.Info("instance", instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			logger.Info("not found")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		logger.Info(" error reading the object")
		return ctrl.Result{}, err
	}
*/

func (r *CSIScaleOperatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// your logic here
	logger := csiLog.WithName("Reconcile")
	logger.Info("in Reconcile")

	// Fetch the Memcached instance
	//	instanceOld := &csiv1.CSIScaleOperator{}
	instance := csiscaleoperator.New(&csiv1.CSIScaleOperator{})
	instanceOld := instance.Unwrap()
	err := r.Client.Get(ctx, req.NamespacedName, instanceOld) //memcached)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			logger.Info("Memcached resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		logger.Error(err, "Failed to get Memcached")
		return ctrl.Result{}, err
	}

	err = r.Client.Update(context.TODO(), instanceOld)
	if err != nil {
		//err = fmt.Errorf("failed to update CSIScaleOperator CR: %v", err)
		return ctrl.Result{}, err
	}
	//	return ctrl.Result{}, nil

	if err := r.addFinalizerIfNotPresent(instanceOld); err != nil {
		return ctrl.Result{}, err
	}

	if !instance.GetDeletionTimestamp().IsZero() {
		isFinalizerExists, err := r.hasFinalizer()
		if err != nil {
			return ctrl.Result{}, err
		}

		if !isFinalizerExists {
			return ctrl.Result{}, nil
		}

		if err := r.deleteClusterRolesAndBindings(); err != nil {
			return ctrl.Result{}, err
		}

		/*		if err := r.deleteCSIDriver(instance); err != nil {
				return ctrl.Result{}, err
			}*/

		if err := r.removeFinalizer(); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	//	instance := csiscaleoperator.New(&csiv1.CSIScaleOperator{})
	originalStatus := *instance.Status.DeepCopy()

	// create the resources which never change if not exist
	for _, rec := range []reconciler{
		r.reconcileCSIDriver,
		r.reconcileServiceAccount,
		r.reconcileClusterRole,
		r.reconcileClusterRoleBinding,
	} {
		if err = rec(instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	// sync the resources which change over time
	csiControllerSyncer := clustersyncer.NewCSIControllerSyncer(r.Client, r.Scheme, instance)
	if err := syncer.Sync(context.TODO(), csiControllerSyncer, r.recorder); err != nil {
		return ctrl.Result{}, err
	}

	csiNodeSyncer := clustersyncer.NewCSINodeSyncer(r.Client, r.Scheme, instance, daemonSetRestartedKey, daemonSetRestartedValue)
	if err := syncer.Sync(context.TODO(), csiNodeSyncer, r.recorder); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.updateStatus(instance, originalStatus); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CSIScaleOperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {

	logger := csiLog.WithName("SetupWithManager")

	logger.Info("in SetupWithManager")
	return ctrl.NewControllerManagedBy(mgr).
		For(&csiv1.CSIScaleOperator{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

func Contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func (r *CSIScaleOperatorReconciler) hasFinalizer() (bool, error) {

	//instance := &csiv1.CSIScaleOperator{}
	accessor, finalizerName, err := r.getAccessorAndFinalizerName()
	if err != nil {
		return false, err
	}

	return Contains(accessor.GetFinalizers(), finalizerName), nil
}

func Remove(list []string, s string) []string {
	var newList []string
	for _, v := range list {
		if v != s {
			newList = append(newList, v)
		}
	}
	return newList
}

func (r *CSIScaleOperatorReconciler) removeFinalizer() error {
	logger := csiLog.WithName("removeFinalizer")

	instance := &csiv1.CSIScaleOperator{}
	accessor, finalizerName, err := r.getAccessorAndFinalizerName()
	if err != nil {
		return err
	}

	accessor.SetFinalizers(Remove(accessor.GetFinalizers(), finalizerName))
	if err := r.Client.Update(context.TODO(), instance); err != nil {
		logger.Error(err, "failed to remove", "finalizer", finalizerName, "from", accessor.GetName())
		return err
	}
	return nil
}

func (r *CSIScaleOperatorReconciler) addFinalizerIfNotPresent(instance *csiv1.CSIScaleOperator) error {
	logger := csiLog.WithName("addFinalizerIfNotPresent")

	//	instance := &csiv1.CSIScaleOperator{}
	accessor, finalizerName, err := r.getAccessorAndFinalizerName()
	if err != nil {
		return err
	}

	if !Contains(accessor.GetFinalizers(), finalizerName) {
		logger.Info("adding", "finalizer", finalizerName, "on", accessor.GetName())
		accessor.SetFinalizers(append(accessor.GetFinalizers(), finalizerName))

		if err := r.Client.Update(context.TODO(), instance); err != nil {
			logger.Error(err, "failed to add", "finalizer", finalizerName, "on", accessor.GetName())
			return err
		}
	}
	return nil
}

func (r *CSIScaleOperatorReconciler) getAccessorAndFinalizerName() (metav1.Object, string, error) {
	logger := csiLog.WithName("getAccessorAndFinalizerName")

	instance := &csiv1.CSIScaleOperator{}
	lowercaseKind := strings.ToLower(instance.GetObjectKind().GroupVersionKind().Kind)
	finalizerName := fmt.Sprintf("%s.%s", lowercaseKind, "ibm.com")

	accessor, err := meta.Accessor(instance)
	if err != nil {
		logger.Error(err, "failed to get meta information of instance")
		return nil, "", err
	}
	return accessor, finalizerName, nil
}

func (r *CSIScaleOperatorReconciler) deleteClusterRolesAndBindings() error {
	/*	if err := r.deleteClusterRoleBindings(instance); err != nil {
		return err
	}*/

	if err := r.deleteClusterRoles(); err != nil {
		return err
	}
	return nil
}

func (r *CSIScaleOperatorReconciler) deleteClusterRoles() error {

	/*	clusterRoles := r.getClusterRoles(instance)

		for _, cr := range clusterRoles {
			found := &rbacv1.ClusterRole{}
			err := r.Client.Get(context.TODO(), types.NamespacedName{
				Name:      cr.Name,
				Namespace: cr.Namespace,
			}, found)
			if err != nil && errors.IsNotFound(err) {
				continue
			} else if err != nil {
				logger.Error(err, "failed to get ClusterRole", "Name", cr.GetName())
				return err
			} else {
				logger.Info("deleting ClusterRole", "Name", cr.GetName())
				if err := r.Client.Delete(context.TODO(), found); err != nil {
					logger.Error(err, "failed to delete ClusterRole", "Name", cr.GetName())
					return err
				}
			}
		}*/
	return nil
}

func (r *CSIScaleOperatorReconciler) reconcileCSIDriver(instance *csiscaleoperator.CSIScaleOperator) error {
	logger := csiLog.WithName("Reconcile CSIDriver")

	logger.Info("in reconcileCSIDriver")
	//	instance := csiscaleoperator.New(&csiv1.CSIScaleOperator{})
	cd := instance.GenerateCSIDriver()
	found := &storagev1.CSIDriver{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{
		Name:      cd.Name,
		Namespace: "",
	}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new CSIDriver", "Name", cd.GetName())
		err = r.Client.Create(context.TODO(), cd)
		if err != nil {
			return err
		}
	} else if err != nil {
		logger.Error(err, "Failed to get CSIDriver", "Name", cd.GetName())
		return err
	} else {
		// Resource already exists - don't requeue
	}

	logger.Info("exiting reconcileCSIDriver")
	return nil
}

func (r *CSIScaleOperatorReconciler) reconcileServiceAccount(instance *csiscaleoperator.CSIScaleOperator) error {
	logger := csiLog.WithName("Reconcile ServiceAccount")

	logger.Info("in reconcileServiceAccount")
	controller := instance.GenerateControllerServiceAccount()
	node := instance.GenerateNodeServiceAccount()

	controllerServiceAccountName := oconfig.GetNameForResource(oconfig.CSIControllerServiceAccount, instance.Name)
	nodeServiceAccountName := oconfig.GetNameForResource(oconfig.CSINodeServiceAccount, instance.Name)

	for _, sa := range []*corev1.ServiceAccount{
		controller,
		node,
	} {
		if err := controllerutil.SetControllerReference(instance.Unwrap(), sa, r.Scheme); err != nil {
			return err
		}
		found := &corev1.ServiceAccount{}
		err := r.Client.Get(context.TODO(), types.NamespacedName{
			Name:      sa.Name,
			Namespace: sa.Namespace,
		}, found)
		if err != nil && errors.IsNotFound(err) {
			logger.Info("Creating a new ServiceAccount", "Namespace", sa.GetNamespace(), "Name", sa.GetName())
			err = r.Client.Create(context.TODO(), sa)
			if err != nil {
				return err
			}

			nodeDaemonSet, err := r.getNodeDaemonSet(instance)
			if err != nil {
				return err
			}

			if controllerServiceAccountName == sa.Name {
				rErr := r.restartControllerPod(logger, instance)

				if rErr != nil {
					return rErr
				}
			}
			if nodeServiceAccountName == sa.Name {
				logger.Info("node rollout requires restart",
					"DesiredNumberScheduled", nodeDaemonSet.Status.DesiredNumberScheduled,
					"NumberAvailable", nodeDaemonSet.Status.NumberAvailable)
				logger.Info("csi node stopped being ready - restarting it")
				rErr := r.rolloutRestartNode(nodeDaemonSet)

				if rErr != nil {
					return rErr
				}

				daemonSetRestartedKey, daemonSetRestartedValue = r.getRestartedAtAnnotation(nodeDaemonSet.Spec.Template.ObjectMeta.Annotations)
			}
		} else if err != nil {
			logger.Error(err, "Failed to get ServiceAccount", "Name", sa.GetName())
			return err
		} else {
			// Resource already exists - don't requeue
			logger.Info("Skip reconcile: ServiceAccount already exists", "Namespace", sa.GetNamespace(), "Name", sa.GetName())
		}
	}
	logger.Info("exiting reconcileServiceAccount")
	return nil
}

func (r *CSIScaleOperatorReconciler) getNodeDaemonSet(instance *csiscaleoperator.CSIScaleOperator) (*appsv1.DaemonSet, error) {
	node := &appsv1.DaemonSet{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{
		Name:      oconfig.GetNameForResource(oconfig.CSINode, instance.Name),
		Namespace: instance.Namespace,
	}, node)

	return node, err
}

func (r *CSIScaleOperatorReconciler) restartControllerPod(logger logr.Logger, instance *csiscaleoperator.CSIScaleOperator) error {
	controllerPod := &corev1.Pod{}
	controllerStatefulset, err := r.getControllerStatefulSet(instance)
	if err != nil {
		return err
	}

	logger.Info("controller requires restart",
		"ReadyReplicas", controllerStatefulset.Status.ReadyReplicas,
		"Replicas", controllerStatefulset.Status.Replicas)
	logger.Info("restarting csi controller")

	err = r.getControllerPod(controllerStatefulset, controllerPod)
	if errors.IsNotFound(err) {
		return nil
	} else if err != nil {
		logger.Error(err, "failed to get controller pod")
		return err
	}

	return r.restartControllerPodfromStatefulSet(logger, controllerStatefulset, controllerPod)
}

func (r *CSIScaleOperatorReconciler) getControllerPod(controllerStatefulset *appsv1.StatefulSet, controllerPod *corev1.Pod) error {
	controllerPodName := fmt.Sprintf("%s-0", controllerStatefulset.Name)
	err := r.Client.Get(context.TODO(), types.NamespacedName{
		Name:      controllerPodName,
		Namespace: controllerStatefulset.Namespace,
	}, controllerPod)
	if errors.IsNotFound(err) {
		return nil
	}
	return err
}

func (r *CSIScaleOperatorReconciler) restartControllerPodfromStatefulSet(logger logr.Logger,
	controllerStatefulset *appsv1.StatefulSet, controllerPod *corev1.Pod) error {
	logger.Info("controller requires restart",
		"ReadyReplicas", controllerStatefulset.Status.ReadyReplicas,
		"Replicas", controllerStatefulset.Status.Replicas)
	logger.Info("restarting csi controller")

	return r.Client.Delete(context.TODO(), controllerPod)
}

func (r *CSIScaleOperatorReconciler) rolloutRestartNode(node *appsv1.DaemonSet) error {
	restartedAt := fmt.Sprintf("%s/restartedAt", oconfig.APIGroup)
	timestamp := time.Now().String()
	node.Spec.Template.ObjectMeta.Annotations[restartedAt] = timestamp
	return r.Client.Update(context.TODO(), node)
}

func (r *CSIScaleOperatorReconciler) getRestartedAtAnnotation(Annotations map[string]string) (string, string) {
	restartedAt := fmt.Sprintf("%s/restartedAt", oconfig.APIGroup)
	for key, element := range Annotations {
		if key == restartedAt {
			return key, element
		}
	}
	return "", ""
}

func (r *CSIScaleOperatorReconciler) getControllerStatefulSet(instance *csiscaleoperator.CSIScaleOperator) (*appsv1.StatefulSet, error) {
	controllerStatefulset := &appsv1.StatefulSet{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{
		Name:      oconfig.GetNameForResource(oconfig.CSIController, instance.Name),
		Namespace: instance.Namespace,
	}, controllerStatefulset)

	return controllerStatefulset, err
}

func (r *CSIScaleOperatorReconciler) reconcileClusterRole(instance *csiscaleoperator.CSIScaleOperator) error {
	logger := csiLog.WithValues("Resource Type", "ClusterRole")

	logger.Info("in reconcileClusterRole")
	clusterRoles := r.getClusterRoles(instance)

	for _, cr := range clusterRoles {
		found := &rbacv1.ClusterRole{}
		err := r.Client.Get(context.TODO(), types.NamespacedName{
			Name:      cr.Name,
			Namespace: cr.Namespace,
		}, found)
		if err != nil && errors.IsNotFound(err) {
			logger.Info("Creating a new ClusterRole", "Name", cr.GetName())
			err = r.Client.Create(context.TODO(), cr)
			if err != nil {
				return err
			}
		} else if err != nil {
			logger.Error(err, "Failed to get ClusterRole", "Name", cr.GetName())
			return err
		} else {
			err = r.Client.Update(context.TODO(), cr)
			if err != nil {
				logger.Error(err, "Failed to update ClusterRole", "Name", cr.GetName())
				return err
			}
		}
	}

	logger.Info("in reconcileClusterRole")
	return nil
}

func (r *CSIScaleOperatorReconciler) getClusterRoles(instance *csiscaleoperator.CSIScaleOperator) []*rbacv1.ClusterRole {
	externalProvisioner := instance.GenerateExternalProvisionerClusterRole()
	externalAttacher := instance.GenerateExternalAttacherClusterRole()
	externalSnapshotter := instance.GenerateExternalSnapshotterClusterRole()
	externalResizer := instance.GenerateExternalResizerClusterRole()
	controllerSCC := instance.GenerateSCCForControllerClusterRole()
	nodeSCC := instance.GenerateSCCForNodeClusterRole()

	return []*rbacv1.ClusterRole{
		externalProvisioner,
		externalAttacher,
		externalSnapshotter,
		externalResizer,
		controllerSCC,
		nodeSCC,
	}
}

func (r *CSIScaleOperatorReconciler) getClusterRoleBindings(instance *csiscaleoperator.CSIScaleOperator) []*rbacv1.ClusterRoleBinding {
	externalProvisioner := instance.GenerateExternalProvisionerClusterRoleBinding()
	externalAttacher := instance.GenerateExternalAttacherClusterRoleBinding()
	externalSnapshotter := instance.GenerateExternalSnapshotterClusterRoleBinding()
	externalResizer := instance.GenerateExternalResizerClusterRoleBinding()
	controllerSCC := instance.GenerateSCCForControllerClusterRoleBinding()
	nodeSCC := instance.GenerateSCCForNodeClusterRoleBinding()

	return []*rbacv1.ClusterRoleBinding{
		externalProvisioner,
		externalAttacher,
		externalSnapshotter,
		externalResizer,
		controllerSCC,
		nodeSCC,
	}
}

func (r *CSIScaleOperatorReconciler) reconcileClusterRoleBinding(instance *csiscaleoperator.CSIScaleOperator) error {
	logger := csiLog.WithValues("Resource Type", "ClusterRoleBinding")

	logger.Info("in reconcileClusterRoleBinding")
	clusterRoleBindings := r.getClusterRoleBindings(instance)

	for _, crb := range clusterRoleBindings {
		found := &rbacv1.ClusterRoleBinding{}
		err := r.Client.Get(context.TODO(), types.NamespacedName{
			Name:      crb.Name,
			Namespace: crb.Namespace,
		}, found)
		if err != nil && errors.IsNotFound(err) {
			logger.Info("Creating a new ClusterRoleBinding", "Name", crb.GetName())
			err = r.Client.Create(context.TODO(), crb)
			if err != nil {
				return err
			}
		} else if err != nil {
			logger.Error(err, "Failed to get ClusterRole", "Name", crb.GetName())
			return err
		} else {
			// Resource already exists - don't requeue
			//logger.Info("Skip reconcile: ClusterRoleBinding already exists", "Name", crb.GetName())
		}
	}

	logger.Info("exiting reconcileClusterRoleBinding")
	return nil
}

func (r *CSIScaleOperatorReconciler) updateStatus(instance *csiscaleoperator.CSIScaleOperator, originalStatus csiv1.CSIScaleOperatorStatus) error {
	logger := csiLog.WithName("updateStatus")
	controllerPod := &corev1.Pod{}
	controllerStatefulset, err := r.getControllerStatefulSet(instance)
	if err != nil {
		return err
	}

	nodeDaemonSet, err := r.getNodeDaemonSet(instance)
	if err != nil {
		return err
	}

	instance.Status.ControllerReady = r.isControllerReady(controllerStatefulset)
	instance.Status.NodeReady = r.isNodeReady(nodeDaemonSet)
	phase := csiv1.DriverPhaseNone
	if instance.Status.ControllerReady && instance.Status.NodeReady {
		phase = csiv1.DriverPhaseRunning
	} else {
		if !instance.Status.ControllerReady {
			err := r.getControllerPod(controllerStatefulset, controllerPod)
			if err != nil {
				logger.Error(err, "failed to get controller pod")
				return err
			}

			if !r.areAllPodImagesSynced(controllerStatefulset, controllerPod) {
				r.restartControllerPodfromStatefulSet(logger, controllerStatefulset, controllerPod)
			}
		}
		phase = csiv1.DriverPhaseCreating
	}
	instance.Status.Phase = phase
	instance.Status.Version = oversion.DriverVersion

	if !reflect.DeepEqual(originalStatus, instance.Status) {
		logger.Info("updating CSIScaleOperator status", "name", instance.Name, "from", originalStatus, "to", instance.Status)
		sErr := r.Client.Status().Update(context.TODO(), instance.Unwrap())
		if sErr != nil {
			return sErr
		}
	}

	return nil
}

func (r *CSIScaleOperatorReconciler) isControllerReady(controller *appsv1.StatefulSet) bool {
	return controller.Status.ReadyReplicas == controller.Status.Replicas
}

func (r *CSIScaleOperatorReconciler) isNodeReady(node *appsv1.DaemonSet) bool {
	return node.Status.DesiredNumberScheduled == node.Status.NumberAvailable
}

func (r *CSIScaleOperatorReconciler) areAllPodImagesSynced(controllerStatefulset *appsv1.StatefulSet, controllerPod *corev1.Pod) bool {
	logger := csiLog.WithName("areAllPodImagesSynced")
	statefulSetContainers := controllerStatefulset.Spec.Template.Spec.Containers
	podContainers := controllerPod.Spec.Containers
	if len(statefulSetContainers) != len(podContainers) {
		return false
	}
	for i := 0; i < len(statefulSetContainers); i++ {
		statefulSetImage := statefulSetContainers[i].Image
		podImage := podContainers[i].Image

		if statefulSetImage != podImage {
			logger.Info("csi controller image not in sync",
				"statefulSetImage", statefulSetImage, "podImage", podImage)
			return false
		}
	}
	return true
}
