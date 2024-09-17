/*
Copyright 2024.

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

package controller

import (
	"context"
	"io"
	"strings"

	crdv1alpha1 "norbinto/kube-ncleaner/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// NodeCleanerReconciler reconciles a NodeCleaner object
type NodeCleanerReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	KubeClient *kubernetes.Clientset
}

// +kubebuilder:rbac:groups=crd.norbinto,resources=nodecleaners,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=crd.norbinto,resources=nodecleaners/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=crd.norbinto,resources=nodecleaners/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NodeCleaner object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *NodeCleanerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("controller is doing magic")

	var nodeCleaner crdv1alpha1.NodeCleaner
	if err := r.Get(ctx, req.NamespacedName, &nodeCleaner); err != nil {
		logger.Error(err, "unable to fetch nodecleaner")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Define a PodList object to hold the list of pods
	podList := &corev1.PodList{}

	// Use the client to list all pods
	if err := r.Client.List(ctx, podList, &client.ListOptions{}); err != nil {
		logger.Error(err, "Failed to list pods")
		return ctrl.Result{}, err
	}

	// Iterate over the pods and filter for running ones
	runningPods := []corev1.Pod{}
	for _, pod := range podList.Items {
		if pod.Status.Phase == corev1.PodRunning {
			runningPods = append(runningPods, pod)
		}
	}

	opts := corev1.PodLogOptions{}
	// Log the running pods
	for _, pod := range runningPods {

		if req.Namespace != pod.Namespace {
			continue
		}

		podLogStream := r.KubeClient.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &opts)

		podLogs, err := podLogStream.Stream(ctx)
		if err != nil {
			return ctrl.Result{}, err
		}
		defer podLogs.Close()

		logs, err := io.ReadAll(podLogs)
		if err != nil {
			return ctrl.Result{}, err
		}

		if strings.HasSuffix(string(logs), nodeCleaner.Spec.LastLogLine) {
			logger.Info(pod.Name + " pod cen be deleted freely")
		}
	}

	return ctrl.Result{}, nil
}

// // Get constructs a request for getting the logs for a pod
// func (c *pods) x(name string, opts *v1.PodLogOptions) *restclient.Request {
// 	return c.GetClient().Get().Namespace(c.GetNamespace()).Name(name).Resource("pods").SubResource("log").VersionedParams(opts, scheme.ParameterCodec)
// }

// SetupWithManager sets up the controller with the Manager.
func (r *NodeCleanerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&crdv1alpha1.NodeCleaner{}).
		Complete(r)
}
