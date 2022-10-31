/*
Copyright 2014 The Kubernetes Authors.

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

// Package options provides the flags used for the controller manager.
//
package options

import (
	"fmt"
	"net"

	v1 "k8s.io/api/core/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	apiserveroptions "k8s.io/apiserver/pkg/server/options"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	clientset "k8s.io/client-go/kubernetes"
	clientgokubescheme "k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	cpoptions "k8s.io/cloud-provider/options"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"
	logsapi "k8s.io/component-base/logs/api/v1"
	"k8s.io/component-base/metrics"
	cmoptions "k8s.io/controller-manager/options"
	kubectrlmgrconfigv1alpha1 "k8s.io/kube-controller-manager/config/v1alpha1"
	kubecontrollerconfig "k8s.io/kubernetes/cmd/kube-controller-manager/app/config"
	"k8s.io/kubernetes/pkg/cluster/ports"
	kubectrlmgrconfig "k8s.io/kubernetes/pkg/controller/apis/config"
	kubectrlmgrconfigscheme "k8s.io/kubernetes/pkg/controller/apis/config/scheme"
	"k8s.io/kubernetes/pkg/controller/garbagecollector"
	garbagecollectorconfig "k8s.io/kubernetes/pkg/controller/garbagecollector/config"
	netutils "k8s.io/utils/net"

	// add the kubernetes feature gates
	_ "k8s.io/kubernetes/pkg/features"
)

const (
	// KubeControllerManagerUserAgent is the userAgent name when starting kube-controller managers.
	KubeControllerManagerUserAgent = "kube-controller-manager"
)

// KubeControllerManagerOptions is the main context object for the kube-controller manager.
type KubeControllerManagerOptions struct {
	Generic           *cmoptions.GenericControllerManagerConfigurationOptions
	KubeCloudShared   *cpoptions.KubeCloudSharedOptions
	ServiceController *cpoptions.ServiceControllerOptions

	AttachDetachController           *AttachDetachControllerOptions
	CSRSigningController             *CSRSigningControllerOptions
	DaemonSetController              *DaemonSetControllerOptions
	DeploymentController             *DeploymentControllerOptions
	StatefulSetController            *StatefulSetControllerOptions
	DeprecatedFlags                  *DeprecatedControllerOptions
	EndpointController               *EndpointControllerOptions
	EndpointSliceController          *EndpointSliceControllerOptions
	EndpointSliceMirroringController *EndpointSliceMirroringControllerOptions
	EphemeralVolumeController        *EphemeralVolumeControllerOptions
	GarbageCollectorController       *GarbageCollectorControllerOptions
	HPAController                    *HPAControllerOptions
	JobController                    *JobControllerOptions
	CronJobController                *CronJobControllerOptions
	NamespaceController              *NamespaceControllerOptions
	NodeIPAMController               *NodeIPAMControllerOptions
	NodeLifecycleController          *NodeLifecycleControllerOptions
	PersistentVolumeBinderController *PersistentVolumeBinderControllerOptions
	PodGCController                  *PodGCControllerOptions
	ReplicaSetController             *ReplicaSetControllerOptions
	ReplicationController            *ReplicationControllerOptions
	ResourceQuotaController          *ResourceQuotaControllerOptions
	SAController                     *SAControllerOptions
	TTLAfterFinishedController       *TTLAfterFinishedControllerOptions

	SecureServing  *apiserveroptions.SecureServingOptionsWithLoopback
	Authentication *apiserveroptions.DelegatingAuthenticationOptions
	Authorization  *apiserveroptions.DelegatingAuthorizationOptions
	Metrics        *metrics.Options
	Logs           *logs.Options

	Master                      string
	Kubeconfig                  string
	ShowHiddenMetricsForVersion string
}

// NewKubeControllerManagerOptions creates a new KubeControllerManagerOptions with a default config.
// NewKubeControllerManagerOptions 创建一个新的 KubeControllerManagerOptions 对象，使用默认配置
func NewKubeControllerManagerOptions() (*KubeControllerManagerOptions, error) {
	// componentConfig 代表 kube-controller-manager 的默认配置，其中包含了所有的 controller 的默认配置
	componentConfig, err := NewDefaultComponentConfig()
	// 如果出错，则直接返回
	if err != nil {
		return nil, err
	}

	s := KubeControllerManagerOptions{
		// 创建一个新的 GenericControllerManagerConfigurationOptions 对象，该对象包含了 kube-controller-manager 的通用配置
		// 通用配置包括：kubeconfig、master、concurrentEndpointSyncs、concurrentServiceSyncs、concurrentRCSyncs、concurrentRSSyncs、
		// concurrentDaemonSetSyncs、concurrentResourceQuotaSyncs、concurrentDeploymentSyncs、concurrentNamespaceSyncs 等。
		Generic: cmoptions.NewGenericControllerManagerConfigurationOptions(&componentConfig.Generic),
		// 创建一个新的 KubeCloudSharedOptions 对象，该对象用于存储与云相关的配置。云相关配置涉及到的 controller 有：
		// attachDetachController、routeController、serviceController、persistentVolumeLabelController、nodeIpamController 等。
		KubeCloudShared: cpoptions.NewKubeCloudSharedOptions(&componentConfig.KubeCloudShared),
		// 创建一个新的 ServiceControllerOptions 对象，该对象用于存储与 service 相关的配置。service controller 用于处理 service
		// 的创建、更新、删除等操作。重要的配置项包括：concurrentServiceSyncs、clusterName、clusterCIDR、nodeCIDRMaskSize 等
		ServiceController: &cpoptions.ServiceControllerOptions{
			ServiceControllerConfiguration: &componentConfig.ServiceController,
		},
		// 创建一个新的 AttachDetachControllerOptions 对象，该对象用于存储与 attach/detach 相关的配置。attach/detach controller
		// 用于处理 volume 的 attach/detach 操作。重要的配置项包括：reconcilerSyncLoopPeriod、reconcilerSyncLoopPeriod 等
		AttachDetachController: &AttachDetachControllerOptions{
			&componentConfig.AttachDetachController,
		},
		// 创建一个新的 CSRSigningControllerOptions 对象，该对象用于存储与 CSR 相关的配置。CSR controller 用于处理 CSR 的创建、更新、
		// 删除等操作，CSR 指的是 Certificate Signing Request，即证书签名请求。重要的配置项包括：clusterSigningCertFile、等。
		CSRSigningController: &CSRSigningControllerOptions{
			&componentConfig.CSRSigningController,
		},
		// 创建一个新的 DaemonSetControllerOptions 对象，该对象用于存储与 DaemonSet 相关的配置。DaemonSet controller 用于处理 DaemonSet
		// 的创建、更新、删除等操作。重要的配置项包括：concurrentDaemonSetSyncs、daemonSetUpdatePeriod 等。
		DaemonSetController: &DaemonSetControllerOptions{
			&componentConfig.DaemonSetController,
		},
		// 创建一个新的 DeploymentControllerOptions 对象，该对象用于存储与 Deployment 相关的配置。Deployment controller 用于处理 Deployment
		// 的创建、更新、删除等操作。重要的配置项包括：concurrentDeploymentSyncs、deploymentControllerSyncPeriod 等。
		DeploymentController: &DeploymentControllerOptions{
			&componentConfig.DeploymentController,
		},
		// 创建一个新的 StatefulSetController 对象，该对象用于存储与 StatefulSet 相关的配置。StatefulSet controller 用于处理 StatefulSet
		// 的创建、更新、删除等操作。StatefulSet 指的是 StatefulSet，即有状态副本集。重要的配置项包括：concurrentStatefulSetSyncs 等。StatefulSet
		// 在 1.9 版本中才被引入，因此在 1.8 版本中，该配置项是无效的。 重要的配置项包括：concurrentStatefulSetSyncs、statefulSetControllerSyncPeriod 等。
		StatefulSetController: &StatefulSetControllerOptions{
			&componentConfig.StatefulSetController,
		},
		// 创建一个新的 DeprecatedControllerOptions 对象，该对象用于存储与 Deprecated 相关的配置。Deprecated controller 用于处理 Deprecated
		// 的创建、更新、删除等操作。其中 Deprecated 指的是已经废弃的 controller，目前该 controller 为空。
		DeprecatedFlags: &DeprecatedControllerOptions{
			&componentConfig.DeprecatedController,
		},
		// 创建一个新的 EndpointControllerOptions 对象，该对象用于存储与 Endpoint 相关的配置。Endpoint controller 用于处理 Endpoint 的创建、
		// 更新、删除等操作。重要的配置项包括：concurrentEndpointSyncs、concurrentEndpointSyncs 等。
		EndpointController: &EndpointControllerOptions{
			&componentConfig.EndpointController,
		},
		// 创建一个新的 EndpointSliceController 对象，该对象用于存储与 EndpointSlice 相关的配置。EndpointSlice controller 用于处理 EndpointSlice
		// 的创建、更新、删除等操作。重要的配置项包括：concurrentEndpointSliceSyncs、endpointSliceSyncPeriod 等。
		EndpointSliceController: &EndpointSliceControllerOptions{
			&componentConfig.EndpointSliceController,
		},
		// 创建一个新的 EndpointSliceMirroringController 对象，该对象用于存储与 EndpointSliceMirroring 相关的配置。EndpointSliceMirroring 是指
		// EndpointSlice 的镜像，即 EndpointSlice 的副本。EndpointSliceMirroring controller 用于处理 EndpointSliceMirroring 的创建、更新、删除等操作。
		EndpointSliceMirroringController: &EndpointSliceMirroringControllerOptions{
			&componentConfig.EndpointSliceMirroringController,
		},
		// 创建一个新的 EphemeralVolumeController 配置对象，该对象用于存储与 EphemeralVolume 相关的配置。EphemeralVolume 是指临时卷。
		EphemeralVolumeController: &EphemeralVolumeControllerOptions{
			&componentConfig.EphemeralVolumeController,
		},
		// 创建一个新的 GCControllerOptions 对象，该对象用于存储与 GC 相关的配置。GC controller 用于处理 GC 的创建、更新、删除等操作。
		// CG 的全称是 Garbage Collection，即垃圾回收，应用场景是当某个资源被删除时，其关联的资源也会被删除。重要的配置项包括：concurrentGCSyncs 等。
		GarbageCollectorController: &GarbageCollectorControllerOptions{
			&componentConfig.GarbageCollectorController,
		},
		// 创建一个新的 HPAControllerOptions 对象，该对象用于存储与 HPA 相关的配置。HPA controller 用于处理 HPA 的创建、更新、删除等操作。
		// HPA 的全称是 Horizontal Pod Autoscaler，即水平 Pod 自动伸缩器。重要的配置项包括：concurrentHPASyncs 等。
		HPAController: &HPAControllerOptions{
			&componentConfig.HPAController,
		},
		// 创建一个新的 JobControllerOptions 对象，该对象用于存储与 Job 相关的配置。Job controller 用于处理 Job 的创建、更新、删除等操作。
		// Job 的全称是 Job，即作业，应用场景是执行一次性任务。重要的配置项包括：concurrentJobSyncs 等。
		JobController: &JobControllerOptions{
			&componentConfig.JobController,
		},
		// 创建一个新的 CronJobControllerOptions 对象，该对象用于存储与 CronJob 相关的配置。CronJob controller 用于处理 CronJob 的创建、更新、删除等操作。
		// CronJob 的全称是 CronJob，即定时作业，应用场景是在指定的时间点执行某个 Job。重要的配置项包括：concurrentCronJobSyncs 等。
		CronJobController: &CronJobControllerOptions{
			&componentConfig.CronJobController,
		},
		// 创建一个新的 NamespaceControllerOptions 对象，该对象用于存储与 Namespace 相关的配置。Namespace controller 用于处理 Namespace 的创建、更新、删除等操作。
		// Namespace 即命名空间，应用场景是将系统中的资源进行分组。重要的配置项包括：concurrentNamespaceSyncs 等。
		NamespaceController: &NamespaceControllerOptions{
			&componentConfig.NamespaceController,
		},
		// 创建一个新的 NodeIpamControllerOptions 对象，该对象用于存储与 NodeIpam 相关的配置。NodeIpam controller 用于处理 NodeIpam 的创建、更新、删除等操作。
		// NodeIpam 的全称是 Node IPAM，即节点 IPAM，应用场景是为节点分配 IP 地址。重要的配置项包括：concurrentNodeSyncs 等。
		NodeIPAMController: &NodeIPAMControllerOptions{
			&componentConfig.NodeIPAMController,
		},
		// 创建一个新的 NodeLifecycleControllerOptions 对象，该对象用于存储与 NodeLifecycle 相关的配置。NodeLifecycle controller 用于处理 NodeLifecycle 的创建、更新、删除等操作。
		// NodeLifecycle 的全称是 Node Lifecycle，即节点生命周期，应用场景是管理节点的生命周期。重要的配置项包括：concurrentNodeSyncs 等。
		NodeLifecycleController: &NodeLifecycleControllerOptions{
			&componentConfig.NodeLifecycleController,
		},
		// 创建一个新的 PersistentVolumeBinderControllerOptions 对象，该对象用于存储与 PersistentVolumeBinder 相关的配置。PersistentVolumeBinder controller 用于处理 PersistentVolumeBinder 的创建、更新、删除等操作。
		// PersistentVolumeBinder 的全称是 Persistent Volume Binder，即持久卷绑定器，应用场景是将持久卷绑定到持久卷声明。重要的配置项包括：pvClaimBinderSyncPeriod 等。
		PersistentVolumeBinderController: &PersistentVolumeBinderControllerOptions{
			&componentConfig.PersistentVolumeBinderController,
		},
		// 创建一个新的 PodGCControllerOptions 对象，该对象用于存储与 PodGC 相关的配置。PodGC controller 用于处理 PodGC 的创建、更新、删除等操作。
		// PodGC 的全称是 Pod GC，即 Pod 垃圾回收，应用场景是回收已经终止的 Pod。重要的配置项包括：terminatedPodGCThreshold 等。
		PodGCController: &PodGCControllerOptions{
			&componentConfig.PodGCController,
		},
		// 创建一个新的 ReplicationControllerOptions 对象，该对象用于存储与 ReplicationController 相关的配置。ReplicationController controller 用于处理 ReplicationController 的创建、更新、删除等操作。
		// ReplicationController 的全称是 Replication Controller，即复制控制器，应用场景是确保指定数量的 Pod 始终处于运行状态。重要的配置项包括：concurrentRCSyncs 等。
		ReplicaSetController: &ReplicaSetControllerOptions{
			&componentConfig.ReplicaSetController,
		},
		// 创建一个新的 ReplicationControllerOptions 对象，该对象用于存储与 ReplicationController 相关的配置。ReplicationController controller 用于处理 ReplicationController 的创建、更新、删除等操作。
		// ReplicationController 的全称是 Replication Controller，即复制控制器，应用场景是确保指定数量的 Pod 始终处于运行状态。重要的配置项包括：concurrentRCSyncs 等。
		ReplicationController: &ReplicationControllerOptions{
			&componentConfig.ReplicationController,
		},
		// 创建一个新的 ResourceQuotaControllerOptions 对象，该对象用于存储与 ResourceQuota 相关的配置。ResourceQuota controller 用于处理 ResourceQuota 的创建、更新、删除等操作。
		// ResourceQuota 的全称是 Resource Quota，即资源配额，应用场景是限制命名空间的资源使用情况。重要的配置项包括：concurrentResourceQuotaSyncs 等。
		ResourceQuotaController: &ResourceQuotaControllerOptions{
			&componentConfig.ResourceQuotaController,
		},
		// 创建一个新的 SAControllerOptions 对象，该对象用于存储与 SA 相关的配置。SA controller 用于处理 SA 的创建、更新、删除等操作。
		// SA 的全称是 Service Account，即服务账号，应用场景是为新的命名空间创建默认的服务账号。重要的配置项包括：concurrentSATokenSyncs 等。
		SAController: &SAControllerOptions{
			&componentConfig.SAController,
		},
		// 创建一个新的 TTLAfterFinishedControllerOptions 对象，该对象用于存储与 TTLAfterFinished 相关的配置。TTLAfterFinished controller 用于处理 TTLAfterFinished 的创建、更新、删除等操作。
		// TTLAfterFinished 的全称是 TTL After Finished，即完成后的 TTL，应用场景是在 Pod 完成后，根据设置的 TTL 时间删除该 Pod。重要的配置项包括：concurrentTTLSyncs 等。
		TTLAfterFinishedController: &TTLAfterFinishedControllerOptions{
			&componentConfig.TTLAfterFinishedController,
		},
		// SecureServing 是一个安全服务配置对象，用于配置安全服务。
		SecureServing: apiserveroptions.NewSecureServingOptions().WithLoopback(),
		// Authentication 是一个认证配置对象，用于配置认证。
		Authentication: apiserveroptions.NewDelegatingAuthenticationOptions(),
		// Authorization 是一个授权配置对象，用于配置授权。
		Authorization: apiserveroptions.NewDelegatingAuthorizationOptions(),
		// Metrics 是一个指标配置对象，用于配置指标。
		Metrics: metrics.NewOptions(),
		// Logs 是一个日志配置对象，用于配置日志。
		Logs: logs.NewOptions(),
	}

	// Authentication.RemoteKubeConfigFileOptional 是否允许远程 KubeConfig 文件为空，该配置项用于AuthenticationOptions的初始化。
	s.Authentication.RemoteKubeConfigFileOptional = true
	// Authorization.RemoteKubeConfigFileOptional 是否允许远程 KubeConfig 文件为空，该配置项用于AuthorizationOptions的初始化。
	s.Authorization.RemoteKubeConfigFileOptional = true

	// Set the PairName but leave certificate directory blank to generate in-memory by default
	// 设置证书对名称，但不设置证书目录，以默认方式在内存中生成。
	s.SecureServing.ServerCert.CertDirectory = ""
	// PairName 是证书对名称，该配置项用于SecureServingOptions的初始化。
	s.SecureServing.ServerCert.PairName = "kube-controller-manager"
	// BindPort 是监听端口，该配置项用于SecureServingOptions的初始化。
	s.SecureServing.BindPort = ports.KubeControllerManagerPort

	// gcIgnoredResources 是垃圾回收忽略的资源，该配置项用于GarbageCollectorControllerOptions的初始化。
	gcIgnoredResources := make([]garbagecollectorconfig.GroupResource, 0, len(garbagecollector.DefaultIgnoredResources()))
	// 遍历垃圾回收默认忽略的资源。
	for r := range garbagecollector.DefaultIgnoredResources() {
		gcIgnoredResources = append(gcIgnoredResources, garbagecollectorconfig.GroupResource{Group: r.Group, Resource: r.Resource})
	}
	// 设置垃圾回收忽略的资源。
	s.GarbageCollectorController.GCIgnoredResources = gcIgnoredResources
	// LeaderElection.ResourceName 是领导者选举资源名称，该配置项用于LeaderElectionConfiguration的初始化。
	s.Generic.LeaderElection.ResourceName = "kube-controller-manager"
	// LeaderElection.ResourceNamespace 是领导者选举资源命名空间，该配置项用于LeaderElectionConfiguration的初始化。
	s.Generic.LeaderElection.ResourceNamespace = "kube-system"

	return &s, nil
}

// NewDefaultComponentConfig returns kube-controller manager configuration object.
func NewDefaultComponentConfig() (kubectrlmgrconfig.KubeControllerManagerConfiguration, error) {
	versioned := kubectrlmgrconfigv1alpha1.KubeControllerManagerConfiguration{}
	kubectrlmgrconfigscheme.Scheme.Default(&versioned)

	internal := kubectrlmgrconfig.KubeControllerManagerConfiguration{}
	if err := kubectrlmgrconfigscheme.Scheme.Convert(&versioned, &internal, nil); err != nil {
		return internal, err
	}
	return internal, nil
}

// Flags returns flags for a specific APIServer by section name
func (s *KubeControllerManagerOptions) Flags(allControllers []string, disabledByDefaultControllers []string) cliflag.NamedFlagSets {
	fss := cliflag.NamedFlagSets{}
	s.Generic.AddFlags(&fss, allControllers, disabledByDefaultControllers)
	s.KubeCloudShared.AddFlags(fss.FlagSet("generic"))
	s.ServiceController.AddFlags(fss.FlagSet("service controller"))

	s.SecureServing.AddFlags(fss.FlagSet("secure serving"))
	s.Authentication.AddFlags(fss.FlagSet("authentication"))
	s.Authorization.AddFlags(fss.FlagSet("authorization"))

	s.AttachDetachController.AddFlags(fss.FlagSet("attachdetach controller"))
	s.CSRSigningController.AddFlags(fss.FlagSet("csrsigning controller"))
	s.DeploymentController.AddFlags(fss.FlagSet("deployment controller"))
	s.StatefulSetController.AddFlags(fss.FlagSet("statefulset controller"))
	s.DaemonSetController.AddFlags(fss.FlagSet("daemonset controller"))
	s.DeprecatedFlags.AddFlags(fss.FlagSet("deprecated"))
	s.EndpointController.AddFlags(fss.FlagSet("endpoint controller"))
	s.EndpointSliceController.AddFlags(fss.FlagSet("endpointslice controller"))
	s.EndpointSliceMirroringController.AddFlags(fss.FlagSet("endpointslicemirroring controller"))
	s.EphemeralVolumeController.AddFlags(fss.FlagSet("ephemeralvolume controller"))
	s.GarbageCollectorController.AddFlags(fss.FlagSet("garbagecollector controller"))
	s.HPAController.AddFlags(fss.FlagSet("horizontalpodautoscaling controller"))
	s.JobController.AddFlags(fss.FlagSet("job controller"))
	s.CronJobController.AddFlags(fss.FlagSet("cronjob controller"))
	s.NamespaceController.AddFlags(fss.FlagSet("namespace controller"))
	s.NodeIPAMController.AddFlags(fss.FlagSet("nodeipam controller"))
	s.NodeLifecycleController.AddFlags(fss.FlagSet("nodelifecycle controller"))
	s.PersistentVolumeBinderController.AddFlags(fss.FlagSet("persistentvolume-binder controller"))
	s.PodGCController.AddFlags(fss.FlagSet("podgc controller"))
	s.ReplicaSetController.AddFlags(fss.FlagSet("replicaset controller"))
	s.ReplicationController.AddFlags(fss.FlagSet("replicationcontroller"))
	s.ResourceQuotaController.AddFlags(fss.FlagSet("resourcequota controller"))
	s.SAController.AddFlags(fss.FlagSet("serviceaccount controller"))
	s.TTLAfterFinishedController.AddFlags(fss.FlagSet("ttl-after-finished controller"))
	s.Metrics.AddFlags(fss.FlagSet("metrics"))
	logsapi.AddFlags(s.Logs, fss.FlagSet("logs"))

	fs := fss.FlagSet("misc")
	fs.StringVar(&s.Master, "master", s.Master, "The address of the Kubernetes API server (overrides any value in kubeconfig).")
	fs.StringVar(&s.Kubeconfig, "kubeconfig", s.Kubeconfig, "Path to kubeconfig file with authorization and master location information.")
	utilfeature.DefaultMutableFeatureGate.AddFlag(fss.FlagSet("generic"))

	return fss
}

// ApplyTo fills up controller manager config with options.
func (s *KubeControllerManagerOptions) ApplyTo(c *kubecontrollerconfig.Config) error {
	if err := s.Generic.ApplyTo(&c.ComponentConfig.Generic); err != nil {
		return err
	}
	if err := s.KubeCloudShared.ApplyTo(&c.ComponentConfig.KubeCloudShared); err != nil {
		return err
	}
	if err := s.AttachDetachController.ApplyTo(&c.ComponentConfig.AttachDetachController); err != nil {
		return err
	}
	if err := s.CSRSigningController.ApplyTo(&c.ComponentConfig.CSRSigningController); err != nil {
		return err
	}
	if err := s.DaemonSetController.ApplyTo(&c.ComponentConfig.DaemonSetController); err != nil {
		return err
	}
	if err := s.DeploymentController.ApplyTo(&c.ComponentConfig.DeploymentController); err != nil {
		return err
	}
	if err := s.StatefulSetController.ApplyTo(&c.ComponentConfig.StatefulSetController); err != nil {
		return err
	}
	if err := s.DeprecatedFlags.ApplyTo(&c.ComponentConfig.DeprecatedController); err != nil {
		return err
	}
	if err := s.EndpointController.ApplyTo(&c.ComponentConfig.EndpointController); err != nil {
		return err
	}
	if err := s.EndpointSliceController.ApplyTo(&c.ComponentConfig.EndpointSliceController); err != nil {
		return err
	}
	if err := s.EndpointSliceMirroringController.ApplyTo(&c.ComponentConfig.EndpointSliceMirroringController); err != nil {
		return err
	}
	if err := s.EphemeralVolumeController.ApplyTo(&c.ComponentConfig.EphemeralVolumeController); err != nil {
		return err
	}
	if err := s.GarbageCollectorController.ApplyTo(&c.ComponentConfig.GarbageCollectorController); err != nil {
		return err
	}
	if err := s.HPAController.ApplyTo(&c.ComponentConfig.HPAController); err != nil {
		return err
	}
	if err := s.JobController.ApplyTo(&c.ComponentConfig.JobController); err != nil {
		return err
	}
	if err := s.CronJobController.ApplyTo(&c.ComponentConfig.CronJobController); err != nil {
		return err
	}
	if err := s.NamespaceController.ApplyTo(&c.ComponentConfig.NamespaceController); err != nil {
		return err
	}
	if err := s.NodeIPAMController.ApplyTo(&c.ComponentConfig.NodeIPAMController); err != nil {
		return err
	}
	if err := s.NodeLifecycleController.ApplyTo(&c.ComponentConfig.NodeLifecycleController); err != nil {
		return err
	}
	if err := s.PersistentVolumeBinderController.ApplyTo(&c.ComponentConfig.PersistentVolumeBinderController); err != nil {
		return err
	}
	if err := s.PodGCController.ApplyTo(&c.ComponentConfig.PodGCController); err != nil {
		return err
	}
	if err := s.ReplicaSetController.ApplyTo(&c.ComponentConfig.ReplicaSetController); err != nil {
		return err
	}
	if err := s.ReplicationController.ApplyTo(&c.ComponentConfig.ReplicationController); err != nil {
		return err
	}
	if err := s.ResourceQuotaController.ApplyTo(&c.ComponentConfig.ResourceQuotaController); err != nil {
		return err
	}
	if err := s.SAController.ApplyTo(&c.ComponentConfig.SAController); err != nil {
		return err
	}
	if err := s.ServiceController.ApplyTo(&c.ComponentConfig.ServiceController); err != nil {
		return err
	}
	if err := s.TTLAfterFinishedController.ApplyTo(&c.ComponentConfig.TTLAfterFinishedController); err != nil {
		return err
	}
	if err := s.SecureServing.ApplyTo(&c.SecureServing, &c.LoopbackClientConfig); err != nil {
		return err
	}
	if s.SecureServing.BindPort != 0 || s.SecureServing.Listener != nil {
		if err := s.Authentication.ApplyTo(&c.Authentication, c.SecureServing, nil); err != nil {
			return err
		}
		if err := s.Authorization.ApplyTo(&c.Authorization); err != nil {
			return err
		}
	}
	return nil
}

// Validate is used to validate the options and config before launching the controller manager
func (s *KubeControllerManagerOptions) Validate(allControllers []string, disabledByDefaultControllers []string) error {
	var errs []error

	errs = append(errs, s.Generic.Validate(allControllers, disabledByDefaultControllers)...)
	errs = append(errs, s.KubeCloudShared.Validate()...)
	errs = append(errs, s.AttachDetachController.Validate()...)
	errs = append(errs, s.CSRSigningController.Validate()...)
	errs = append(errs, s.DaemonSetController.Validate()...)
	errs = append(errs, s.DeploymentController.Validate()...)
	errs = append(errs, s.StatefulSetController.Validate()...)
	errs = append(errs, s.DeprecatedFlags.Validate()...)
	errs = append(errs, s.EndpointController.Validate()...)
	errs = append(errs, s.EndpointSliceController.Validate()...)
	errs = append(errs, s.EndpointSliceMirroringController.Validate()...)
	errs = append(errs, s.EphemeralVolumeController.Validate()...)
	errs = append(errs, s.GarbageCollectorController.Validate()...)
	errs = append(errs, s.HPAController.Validate()...)
	errs = append(errs, s.JobController.Validate()...)
	errs = append(errs, s.CronJobController.Validate()...)
	errs = append(errs, s.NamespaceController.Validate()...)
	errs = append(errs, s.NodeIPAMController.Validate()...)
	errs = append(errs, s.NodeLifecycleController.Validate()...)
	errs = append(errs, s.PersistentVolumeBinderController.Validate()...)
	errs = append(errs, s.PodGCController.Validate()...)
	errs = append(errs, s.ReplicaSetController.Validate()...)
	errs = append(errs, s.ReplicationController.Validate()...)
	errs = append(errs, s.ResourceQuotaController.Validate()...)
	errs = append(errs, s.SAController.Validate()...)
	errs = append(errs, s.ServiceController.Validate()...)
	errs = append(errs, s.TTLAfterFinishedController.Validate()...)
	errs = append(errs, s.SecureServing.Validate()...)
	errs = append(errs, s.Authentication.Validate()...)
	errs = append(errs, s.Authorization.Validate()...)
	errs = append(errs, s.Metrics.Validate()...)

	// TODO: validate component config, master and kubeconfig

	return utilerrors.NewAggregate(errs)
}

// Config return a controller manager config objective
func (s KubeControllerManagerOptions) Config(allControllers []string, disabledByDefaultControllers []string) (*kubecontrollerconfig.Config, error) {
	if err := s.Validate(allControllers, disabledByDefaultControllers); err != nil {
		return nil, err
	}

	if err := s.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil, []net.IP{netutils.ParseIPSloppy("127.0.0.1")}); err != nil {
		return nil, fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	kubeconfig, err := clientcmd.BuildConfigFromFlags(s.Master, s.Kubeconfig)
	if err != nil {
		return nil, err
	}
	kubeconfig.DisableCompression = true
	kubeconfig.ContentConfig.AcceptContentTypes = s.Generic.ClientConnection.AcceptContentTypes
	kubeconfig.ContentConfig.ContentType = s.Generic.ClientConnection.ContentType
	kubeconfig.QPS = s.Generic.ClientConnection.QPS
	kubeconfig.Burst = int(s.Generic.ClientConnection.Burst)

	client, err := clientset.NewForConfig(restclient.AddUserAgent(kubeconfig, KubeControllerManagerUserAgent))
	if err != nil {
		return nil, err
	}

	eventBroadcaster := record.NewBroadcaster()
	eventRecorder := eventBroadcaster.NewRecorder(clientgokubescheme.Scheme, v1.EventSource{Component: KubeControllerManagerUserAgent})

	c := &kubecontrollerconfig.Config{
		Client:           client,
		Kubeconfig:       kubeconfig,
		EventBroadcaster: eventBroadcaster,
		EventRecorder:    eventRecorder,
	}
	if err := s.ApplyTo(c); err != nil {
		return nil, err
	}
	s.Metrics.Apply()

	return c, nil
}
