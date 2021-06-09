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
	"strings"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	csiv1 "github.com/IBM/ibm-spectrum-scale-csi/api/v1"
)

// CSIScaleOperatorReconciler reconciles a CSIScaleOperator object
type CSIScaleOperatorReconciler struct {
	Client client.Client
	Scheme *runtime.Scheme
}

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
func (r *CSIScaleOperatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// your logic here
	instance := &csiv1.CSIScaleOperator{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}


	err = r.Client.Update(context.TODO(), instance)
	if err != nil {
		//err = fmt.Errorf("failed to update IBMBlockCSI CR: %v", err)
		return ctrl.Result{}, err
	}
//	return ctrl.Result{}, nil

	if err := r.addFinalizerIfNotPresent(); err != nil {
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

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CSIScaleOperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&csiv1.CSIScaleOperator{}).
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

	instance := &csiv1.CSIScaleOperator{}
	accessor, finalizerName, err := r.getAccessorAndFinalizerName()
	if err != nil {
		return err
	}

	accessor.SetFinalizers(Remove(accessor.GetFinalizers(), finalizerName))
	if err := r.Client.Update(context.TODO(), instance); err != nil {
//		logger.Error(err, "failed to remove", "finalizer", finalizerName, "from", accessor.GetName())
		return err
	}
	return nil
}

func (r *CSIScaleOperatorReconciler) addFinalizerIfNotPresent() error {

	instance := &csiv1.CSIScaleOperator{}
	accessor, finalizerName, err := r.getAccessorAndFinalizerName()
	if err != nil {
		return err
	}

	if !Contains(accessor.GetFinalizers(), finalizerName) {
		//logger.Info("adding", "finalizer", finalizerName, "on", accessor.GetName())
		accessor.SetFinalizers(append(accessor.GetFinalizers(), finalizerName))

		if err := r.Client.Update(context.TODO(), instance); err != nil {
		//	logger.Error(err, "failed to add", "finalizer", finalizerName, "on", accessor.GetName())
			return err
		}
	}
	return nil
}

func (r *CSIScaleOperatorReconciler) getAccessorAndFinalizerName() (metav1.Object, string, error) {

	instance := &csiv1.CSIScaleOperator{}
	lowercaseKind := strings.ToLower(instance.GetObjectKind().GroupVersionKind().Kind)
	finalizerName := fmt.Sprintf("%s.%s", lowercaseKind, "ibm.com")

	accessor, err := meta.Accessor(instance)
	if err != nil {
//		logger.Error(err, "failed to get meta information of instance")
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
		err := r.client.Get(context.TODO(), types.NamespacedName{
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
			if err := r.client.Delete(context.TODO(), found); err != nil {
				logger.Error(err, "failed to delete ClusterRole", "Name", cr.GetName())
				return err
			}
		}
	}*/
	return nil
}

