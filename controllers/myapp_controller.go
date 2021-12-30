/*
Copyright 2021 jude.

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
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	appv1beta1 "github.com/haozi4263/opdemo/api/v1beta1"
)

var (
	oldSpecAnnotation = "old/spec"
)

// MyAppReconciler reconciles a MyApp object
type MyAppReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// Controller Rbac in Cluster 需要增加Deploymnet Service的权限 (属于core,group为空)
// +kubebuilder:rbac:groups=apps,resources=deploymnets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete

// +kubebuilder:rbac:groups=app.shimo.im,resources=myapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rb ac:groups=app.shimo.im,resources=myapps/status,verbs=get;update;patch

func (r *MyAppReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("myapp", req.NamespacedName)

	//首先获取MyApp实例
	var myapp appv1beta1.MyApp
	if err := r.Client.Get(ctx, req.NamespacedName, &myapp); err != nil {
		// MyApp was deleted,Ingore
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 优化版本逻辑 得到MyApp过后去创建对应的Deploymnet和Service
	// 创建就得判断是否存在，存在就忽略，不存在就创建 还有更新 （就是观察当前状态和期望的状态进行对比）

	// 调谐，获取到当前的一个状态，然后和我们期望的状态进行对比
	// 直接获取到deployment/svc去查询，存在不处理，不存在去创建。还需要判断是否更新
	// CreatOrUpdate会自动帮助我们判断是否创建或更新

	// CreateOrUpdate Deployment
	var deploy appsv1.Deployment
	deploy.Name = myapp.Name // 需要根据当前的name ns查账当前的deployment是否存在
	deploy.Namespace = myapp.Namespace

	//CreateOrUpdate 创建或更新一个给定的k8s对象，用这个给定的状态对这个对象进行期望的状态调谐，需要传递一个回调函数
	or, err := ctrl.CreateOrUpdate(ctx, r, &deploy, func() error {
		// 调谐必须在这个函数中实现 newDeploy
		MutateDeployment(&myapp, &deploy)
		return controllerutil.SetControllerReference(&myapp, &deploy, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	log.Info("CreateOrUpdate", "Deployment", or)

	// CreateOrUpdate Service
	var svc corev1.Service
	svc.Name = myapp.Name
	svc.Namespace = myapp.Namespace
	or, err = ctrl.CreateOrUpdate(ctx, r, &svc, func() error {
		// 调谐必须在这个函数中实现 newService
		MutateService(&myapp, &svc)
		return controllerutil.SetControllerReference(&myapp, &svc, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	log.Info("CreateOrUpdate", "Service", or)

	return ctrl.Result{}, nil
}

func (r *MyAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1beta1.MyApp{}).
		Owns(&appsv1.Deployment{}). // Watche Deploymnet Service属于MyApp资源
		Owns(&corev1.Service{}).    // 实现Service Deployment被删除时候能被Watch到Reconcile
		Complete(r)
}
