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
	"fmt"
	"os"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	skopeoiov1alpha1 "github.com/Tchoupinax/skopeo.io/api/v1alpha1"
)

// ImageReconciler reconciles a Image object
type ImageReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=skopeo.io,resources=images,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=skopeo.io,resources=images/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=skopeo.io,resources=images/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Image object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *ImageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var image skopeoiov1alpha1.Image
	if err := r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, &image); err != nil {
		return ctrl.Result{}, err
	}

	if len(image.Status.History) == 0 {
		image.Status.History = append(image.Status.History, skopeoiov1alpha1.History{
			PerformedAt: metav1.Now(),
		})

		updateError := r.Status().Update(ctx, &image)
		if updateError != nil {
			fmt.Println(updateError)
		}

		createSkopeoPod(r, ctx, req, image)

	} else {
		lastApplication := image.Status.History[len(image.Status.History)-1].PerformedAt
		timeDifference := metav1.Now().Sub(lastApplication.Time)

		fmt.Printf("difference %v", timeDifference)
	}

	if image.Spec.Mode == "OneShot" {
		return ctrl.Result{}, nil
	}

	return ctrl.Result{
		Requeue:      true,
		RequeueAfter: 5 * time.Second,
	}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ImageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&skopeoiov1alpha1.Image{}).
		Complete(r)
}

func createSkopeoPod(r *ImageReconciler, ctx context.Context, req ctrl.Request, image skopeoiov1alpha1.Image) {
	fmt.Println(fmt.Printf("Create job to copy image %s:%s", image.Spec.Source.ImageName, image.Spec.Source.ImageVersion))

	arguments := []string{
		"copy",
		fmt.Sprintf("docker://%s:%s", image.Spec.Source.ImageName, image.Spec.Source.ImageVersion),
		fmt.Sprintf("docker://%s:%s", image.Spec.Destination.ImageName, image.Spec.Destination.ImageVersion),
		"--all",
		"--preserve-digests",
	}

	if os.Getenv("CREDS_DESTINATION_USERNAME") != "" && os.Getenv("CREDS_DESTINATION_PASSWORD") != "" {
		arguments = append(
			arguments,
			fmt.Sprintf("--dest-creds=%s:%s", os.Getenv("CREDS_DESTINATION_USERNAME"), os.Getenv("CREDS_DESTINATION_PASSWORD")),
		)
	}

	if os.Getenv("CREDS_SOURCE_USERNAME") != "" && os.Getenv("CREDS_SOURCE_PASSWORD") != "" {
		arguments = append(
			arguments,
			fmt.Sprintf("--dest-creds=%s:%s", os.Getenv("CREDS_SOURCE_USERNAME"), os.Getenv("CREDS_SOURCE_PASSWORD")),
		)
	}

	podNamespace := os.Getenv("PULL_JOB_NAMESPACE")
	if podNamespace == "" {
		podNamespace = "skopeo-operator"
	}

	skopeoImage := os.Getenv("SKOPEO_IMAGE")
	if skopeoImage == "" {
		skopeoImage = "quay.io/containers/skopeo"
	}

	skopeoVersion := os.Getenv("SKOPEO_VERSION")
	if skopeoVersion == "" {
		skopeoVersion = "v1.16.1"
	}

	desiredPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("skopeo-job-copy-%s", strings.ReplaceAll(req.Name, ".", "_")),
			Namespace: podNamespace,
		},
		Spec: corev1.PodSpec{
			RestartPolicy: "Never",
			Containers: []corev1.Container{
				{
					Image:   fmt.Sprintf("%s:%s", skopeoImage, skopeoVersion),
					Name:    "skopeo",
					Command: []string{"skopeo"},
					Args:    arguments,
				},
			},
		},
	}

	_ = r.Create(ctx, &desiredPod)
}
