# Operator

### Mac 安装 operator-sdk

```
#这里使用的是 1.4.2 版本 operator-sdk 作为基础版本开发
curl -LO https://github.com/operator-framework/operator-sdk/releases/download/v1.4.2/operator-sdk_darwin_amd64
chmod +x operator-sdk_darwin_amd64
sudo cp operator-sdk_darwin_amd64 /usr/local/go/bin/operator-sdk

or

brew install operator-sdk
```

### Workflow
Operator SDK 流程总结为6大部分

* 1、创建 operator 项目文件夹并且进入
* 2、初始化项目
* 3、创建 api
* 4、填充代码
* 5、创建 crd 到集群
* 6、部署 operator 项目

### 首先我们要确定自己所写的项目最终效果是什么样子

```
#首先我们要定义一下最终 yaml 效果与要实现的功能
apiVersion: cache.github.com/v1alpha1
kind: AppService
metadata:
  name: nginx-app
spec:
  image: nginx:latest
  ports:
    - port: 80
      targetPort: 80
      nodePort: 30001

#最终目的是只在一个 yaml 里填写 image 与 port 就可以完成 deployment 与 service 的创建
```

### 创建文件夹、初始化、创建 API

```
mkdir operator-demo && cd operator-demo

#这里初始化的 domain 为 Group 后缀，比方说创建 api 选择 group 为 cache 最后会组合成 cache.github.com，不填写默认为 cache.my.domain
operator-sdk init --domain github.com --repo github.com/darianJmy/operator-demo --plugins go.kubebuilder.io/v2

#这边就是创建 api，GKV 分别为 cache.github.com、AppService、v1alpha1
operator-sdk create api --group cache --version v1alpha1 --kind AppService --resource --controller
```
### 查看构建目录

```
tree
.
├── Dockerfile
├── Makefile
├── PROJECT
├── api
│   └── v1alpha1
│       ├── appservice_types.go
│       ├── groupversion_info.go
│       └── zz_generated.deepcopy.go
├── bin
│   └── manager
├── config
│   ├── certmanager
│   │   ├── certificate.yaml
│   │   ├── kustomization.yaml
│   │   └── kustomizeconfig.yaml
│   ├── crd
│   │   ├── kustomization.yaml
│   │   ├── kustomizeconfig.yaml
│   │   └── patches
│   │       ├── cainjection_in_appservices.yaml
│   │       ├── webhook_in_appservices.yaml
│   ├── default
│   │   ├── kustomization.yaml
│   │   ├── manager_auth_proxy_patch.yaml
│   │   ├── manager_webhook_patch.yaml
│   │   └── webhookcainjection_patch.yaml
│   ├── manager
│   │   ├── kustomization.yaml
│   │   └── manager.yaml
│   ├── prometheus
│   │   ├── kustomization.yaml
│   │   └── monitor.yaml
│   ├── rbac
│   │   ├── appservice_editor_role.yaml
│   │   ├── appservice_viewer_role.yaml
│   │   ├── auth_proxy_client_clusterrole.yaml
│   │   ├── auth_proxy_role.yaml
│   │   ├── auth_proxy_role_binding.yaml
│   │   ├── auth_proxy_service.yaml
│   │   ├── kustomization.yaml
│   │   ├── leader_election_role.yaml
│   │   ├── leader_election_role_binding.yaml
│   │   └── role_binding.yaml
│   ├── samples
│   │   ├── cache_v1alpha1_appservice.yaml
│   │   └── kustomization.yaml
│   ├── scorecard
│   │   ├── bases
│   │   │   └── config.yaml
│   │   ├── kustomization.yaml
│   │   └── patches
│   │       ├── basic.config.yaml
│   │       └── olm.config.yaml
│   └── webhook
│       ├── kustomization.yaml
│       ├── kustomizeconfig.yaml
│       └── service.yaml
├── controllers
│   ├── appservice_controller.go
│   └── suite_test.go
├── go.mod
├── go.sum
├── hack
│   └── boilerplate.go.txt
└── main.go

#这里主要文件夹有 api、controllers 文件夹，api文件夹里对应了 Version、Group、Kind，Controller 主要对应了主程序代码
```

### 填充代码

```
#对 api 文件夹里的 appservice_types.go 文件进行填充 yaml 文件 spec 部分所需要的内容

type AppServiceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of AppService. Edit AppService_types.go to remove/update
	Image  string                `json:"image"`
	Ports  []corev1.ServicePort  `json:"ports,omitempty"`
}

# spec 部分填充好了就可以编写主程序的，主要功能就是 Get 获取服务状态，没有就 Create，更新的就 Update

func (r *AppServiceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	Logger := r.Log.WithValues("AppService", req.NamespacedName)
	Logger.Info("Reconciling AppService")

	// your logic here
    // 这边就是需要填充代码的地方

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

        //resources.NewDeploy 是 deployment.go 代码入口 ，resources 为自己编写存放资源的文件夹
        //deployment 与 service 创建的具体代码在 resources 目录下

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
```

### 创建 CRD 到集群

```
make install
```

### 构建代码变成镜像

```
#这里需要注意我们新增了一个 resources 文件夹也要在 dockerfile 中体现出来
docker build -f Dockerfile . -t operator-demo:v0.0.1
```

### 编写 yaml 部署此服务

```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: operator-demo-deployment
  labels:
    app: operator-demo
spec:
  replicas: 1
  selector:
    matchLabels:
      app: operator-demo
  template:
    metadata:
      labels:
        app: operator-demo
    spec:
      containers:
      - name: operator-demo
        image: 10.10.33.31:5000/operator-demo:v0.0.1
```

### 测试功能是正常

```
cat test.yaml

apiVersion: cache.github.com/v1alpha1
kind: AppService
metadata:
  name: nginx-app
spec:
  image: nginx:latest
  ports:
    - port: 80
      targetPort: 80
      nodePort: 30001

kubectl apply -f test.yaml
appservice.cache.github.com/nginx-app created

kubectl get deployment
NAME                  READY   UP-TO-DATE   AVAILABLE   AGE
nginx-app             1/1     1            1           50s

kubectl get svc
NAME         TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)        AGE
nginx-app    NodePort    10.254.106.44   <none>        80:30001/TCP   75s
```
