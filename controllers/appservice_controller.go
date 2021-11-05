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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"github.com/darianJmy/operator-demo/resources"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"reflect"
	cachev1alpha1 "github.com/darianJmy/operator-demo/api/v1alpha1"
)

// AppServiceReconciler reconciles a AppService object
type AppServiceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=cache.github.com,resources=appservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cache.github.com,resources=appservices/status,verbs=get;update;patch

func (r *AppServiceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	Logger := r.Log.WithValues("AppService", req.NamespacedName)
	Logger.Info("Reconciling AppService")

	// your logic here
	instance := &cachev1alpha1.AppService{}
	err := r.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		Logger.Error(err, "failed to get instance from appservice")
		return ctrl.Result{}, err
	}

	deploy := &appsv1.Deployment{}
	if err := r.Get(context.TODO(), req.NamespacedName, deploy); err != nil && errors.IsNotFound(err) {
		deploy := resources.NewDeploy(instance)
		if err := r.Create(context.TODO(),deploy); err != nil {
			Logger.Error(err, "failed create deployment")
			return ctrl.Result{}, err
		}

	}
	service := &corev1.Service{}
	if err := r.Get(context.TODO(), req.NamespacedName, service); err != nil && errors.IsNotFound(err) {
		service := resources.NewService(instance)
		if err := r.Create(context.TODO(), service); err != nil {
			Logger.Error(err, "failed create service")
			return ctrl.Result{}, err
		}
	}

	oldspec := cachev1alpha1.AppServiceSpec{}
	if !reflect.DeepEqual(instance.Spec,oldspec) {
		newDeploy := resources.NewDeploy(instance)
		oldDeploy := &appsv1.Deployment{}
		if err := r.Get(context.TODO(), req.NamespacedName, oldDeploy); err != nil {
			return ctrl.Result{}, err
		}
		oldDeploy.Spec = newDeploy.Spec
		if err := r.Update(context.TODO(), oldDeploy); err != nil {
			Logger.Error(err, "failed update deployment")
			return ctrl.Result{}, err
		}

		newService := resources.NewService(instance)
		oldService := &corev1.Service{}
		if err := r.Get(context.TODO(), req.NamespacedName, oldService); err != nil {
			return ctrl.Result{}, err
		}
		oldService.Spec.Type = newService.Spec.Type
		oldService.Spec.Ports = newService.Spec.Ports
		oldService.Spec.Selector = newService.Spec.Selector
		if err := r.Update(context.TODO(), oldService); err != nil {
			Logger.Error(err, "failed update service")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

func (r *AppServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1alpha1.AppService{}).
		Complete(r)
}
