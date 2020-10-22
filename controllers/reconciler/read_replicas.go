// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package reconciler

import (
	"fmt"
	neo4jv1alpha1 "github.com/wn-doolittle/neo4j-operator/api/v1alpha1"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"reflect"
)

type ReadReplica struct {
}

func (r *ReadReplica) Create(instance *neo4jv1alpha1.Neo4jCluster) (MetaObject, error) {
	if !instance.Spec.IsCausalCluster() {
		// Replicas are only meaningful if we have a core set.
		return nil, nil
	}
	return buildReadReplicas(instance), nil
}

func (r *ReadReplica) Update(instance *neo4jv1alpha1.Neo4jCluster, found runtime.Object) (MetaObject, bool, error) {
	if !instance.Spec.IsCausalCluster() {
		// Replicas are only meaningful if we have a core set.
		return nil, false, nil
	}

	other := found.(*apps.StatefulSet)
	tmp := buildReadReplicas(instance)
	restart := false

	// Override Docker image version
	other.Spec.Template.Spec.Containers[0].Image = tmp.Spec.Template.Spec.Containers[0].Image

	// Override Docker image pull policy
	other.Spec.Template.Spec.Containers[0].ImagePullPolicy = tmp.Spec.Template.Spec.Containers[0].ImagePullPolicy

	// Scale up or down the cluster
	other.Spec.Replicas = &instance.Spec.ReadReplicaServers

	// Updating environment variables
	if !reflect.DeepEqual(tmp.Spec.Template.Spec.Containers[0].Env, other.Spec.Template.Spec.Containers[0].Env) {
		other.Spec.Template.Spec.Containers[0].Env = tmp.Spec.Template.Spec.Containers[0].Env
		restart = true
	}

	// Update CPU and memory resources
	if !equalResources(&tmp.Spec.Template.Spec.Containers[0].Resources, &other.Spec.Template.Spec.Containers[0].Resources) {
		other.Spec.Template.Spec.Containers[0].Resources = tmp.Spec.Template.Spec.Containers[0].Resources
	}

	return other, restart, nil
}

func (r *ReadReplica) GetName(instance *neo4jv1alpha1.Neo4jCluster) string {
	return instance.ReadReplicaName()
}

func (r *ReadReplica) DefaultObject() runtime.Object {
	return &apps.StatefulSet{}
}

func buildReadReplicas(instance *neo4jv1alpha1.Neo4jCluster) *apps.StatefulSet {
	imagePullPolicy := instance.Spec.ImagePullPolicy
	if imagePullPolicy == "" {
		imagePullPolicy = "IfNotPresent"
	}
	limitCpu, _ := resource.ParseQuantity(instance.Spec.Resources.Limits.CPU)
	limitMemory, _ := resource.ParseQuantity(instance.Spec.Resources.Limits.Memory)
	requestCpu, _ := resource.ParseQuantity(instance.Spec.Resources.Requests.CPU)
	requestMemory, _ := resource.ParseQuantity(instance.Spec.Resources.Requests.Memory)
	dataMountPath := "/data"
	if instance.Spec.PersistentStorage != nil {
		if instance.Spec.PersistentStorage.MountPath != "" {
			dataMountPath = instance.Spec.PersistentStorage.MountPath
		}
	}
	defaultLabels := map[string]string{
		"component": instance.LabelComponentName(),
		"role":      "neo4j-replica",
	}
	statefulSet := &apps.StatefulSet{
		ObjectMeta: meta.ObjectMeta{
			Name:      instance.ReadReplicaName(),
			Namespace: instance.Namespace,
			Labels:    defaultLabels,
		},
		Spec: apps.StatefulSetSpec{
			Replicas:            &instance.Spec.ReadReplicaServers,
			PodManagementPolicy: "Parallel",
			UpdateStrategy: apps.StatefulSetUpdateStrategy{
				Type: apps.RollingUpdateStatefulSetStrategyType,
			},
			Selector: &meta.LabelSelector{
				MatchLabels: defaultLabels,
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: meta.ObjectMeta{
					Labels:      defaultLabels,
					Annotations: instance.Spec.PodAnnotations,
				},
				Spec: core.PodSpec{
					ServiceAccountName: instance.ServiceAccountName(),
					// High value permits checkpointing on Neo4j shutdown.  See: https://neo4j.com/developer/kb/checkpointing-and-log-pruning-interactions/
					TerminationGracePeriodSeconds: func(i int64) *int64 { return &i }(300),
					NodeSelector:                  instance.Spec.NodeSelector,
					Containers: []core.Container{
						{
							Name:            "replica",
							Image:           instance.Spec.DockerImage(),
							ImagePullPolicy: core.PullPolicy(imagePullPolicy),
							Env: []core.EnvVar{
								{
									Name:  "DISCOVERY_HOST_PREFIX",
									Value: instance.DiscoveryServiceNamePrefix(),
								},
								{
									Name: "POD_NAMESPACE",
									ValueFrom: &core.EnvVarSource{
										FieldRef: &core.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
								{
									Name:  "NEO4J_causal__clustering_kubernetes_label__selector",
									Value: fmt.Sprintf("neo4j.com/cluster=%s,neo4j.com/role=CORE,neo4j.com/coreindex in (0, 1, 2)", instance.LabelComponentName()),
								},
							},
							EnvFrom: []core.EnvFromSource{
								{
									ConfigMapRef: &core.ConfigMapEnvSource{
										LocalObjectReference: core.LocalObjectReference{
											Name: instance.CommonConfigMapName(),
										},
									},
								},
								{
									ConfigMapRef: &core.ConfigMapEnvSource{
										LocalObjectReference: core.LocalObjectReference{
											Name: instance.ReplicaConfigMapName(),
										},
									},
								},
							},
							ReadinessProbe: &core.Probe{
								Handler: core.Handler{
									TCPSocket: &core.TCPSocketAction{
										Port: intstr.FromInt(7687),
									},
								},
								InitialDelaySeconds: 120,
								TimeoutSeconds:      2,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    3,
							},
							LivenessProbe: &core.Probe{
								Handler: core.Handler{
									TCPSocket: &core.TCPSocketAction{
										Port: intstr.FromInt(7687),
									},
								},
								InitialDelaySeconds: 300,
								TimeoutSeconds:      2,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    3,
							},
							Command: []string{
								"/bin/bash",
								"-c",
								`
export replica_idx=$(hostname | sed 's|.*-||')

# Processes key configuration elements and exports env vars we need.
. /helm-init/init.sh

# These settings are *not* overrideable, because they must match the addresses the
# core members see to avoid akka rejections.
export NEO4J_causal__clustering_discovery__advertised__address=$HOST:5000
export NEO4J_causal__clustering_transaction__advertised__address=$HOST:6000
export NEO4J_causal__clustering_raft__advertised__address=$HOST:7000

echo "Starting Neo4j READ_REPLICA $replica_idx on $HOST"
exec /docker-entrypoint.sh "neo4j"
`,
							},
							Ports: []core.ContainerPort{
								{Name: "tcp-discovery", ContainerPort: 5000, Protocol: "TCP"},
								{Name: "tcp-tx", ContainerPort: 6000, Protocol: "TCP"},
								{Name: "tcp-raft", ContainerPort: 7000, Protocol: "TCP"},
								{Name: "browser-https", ContainerPort: 7473, Protocol: "TCP"},
								{Name: "tcp-browser", ContainerPort: 7474, Protocol: "TCP"},
								{Name: "tcp-bolt", ContainerPort: 7687, Protocol: "TCP"},
							},
							VolumeMounts: []core.VolumeMount{
								{Name: "init-script", MountPath: "/helm-init"},
								{Name: "datadir", MountPath: dataMountPath},
								{Name: "plugins", MountPath: "/plugins"},
							},
							Resources: core.ResourceRequirements{
								Limits: core.ResourceList{
									"cpu":    limitCpu,
									"memory": limitMemory,
								},
								Requests: core.ResourceList{
									"cpu":    requestCpu,
									"memory": requestMemory,
								},
							},
						},
					},
					Volumes: []core.Volume{
						{
							Name: "init-script",
							VolumeSource: core.VolumeSource{
								ConfigMap: &core.ConfigMapVolumeSource{
									LocalObjectReference: core.LocalObjectReference{
										Name: "neo4j-init-script",
									},
								},
							},
						},
						{
							Name:         "plugins",
							VolumeSource: core.VolumeSource{EmptyDir: &core.EmptyDirVolumeSource{}},
						},
					},
				},
			},
		},
	}
	templateSpec := &statefulSet.Spec.Template.Spec
	if instance.Spec.AuthorizationEnabled() {
		secret := core.EnvVar{
			Name: "NEO4J_SECRETS_PASSWORD",
			ValueFrom: &core.EnvVarSource{
				SecretKeyRef: &core.SecretKeySelector{
					LocalObjectReference: core.LocalObjectReference{Name: instance.SecretStoreName()},
					Key:                  "neo4j-password",
				},
			},
		}
		templateSpec.Containers[0].Env = append(templateSpec.Containers[0].Env, secret)
	}
	if instance.Spec.SslCertificates != nil {
		key := core.EnvVar{
			Name:  "SSL_KEY",
			Value: instance.Spec.SslCertificates.PrivateKey,
		}
		certificate := core.EnvVar{
			Name:  "SSL_CERTIFICATE",
			Value: instance.Spec.SslCertificates.PublicCertificate,
		}
		templateSpec.Containers[0].Env = append(templateSpec.Containers[0].Env, key, certificate)
	}
	for k, v := range instance.Spec.ReadReplicaArguments {
		templateSpec.Containers[0].Env = append(templateSpec.Containers[0].Env, core.EnvVar{Name: k, Value: v})
	}
	if instance.Spec.PersistentStorage == nil {
		templateSpec.Volumes = append(templateSpec.Volumes, core.Volume{
			Name:         "datadir",
			VolumeSource: core.VolumeSource{EmptyDir: &core.EmptyDirVolumeSource{}},
		})
	} else {
		storageSettings := instance.Spec.PersistentStorage
		volumeSize, _ := resource.ParseQuantity(storageSettings.Size)
		statefulSet.Spec.VolumeClaimTemplates = []core.PersistentVolumeClaim{
			{
				ObjectMeta: meta.ObjectMeta{
					Name:   "datadir",
					Labels: defaultLabels,
				},
				Spec: core.PersistentVolumeClaimSpec{
					AccessModes: []core.PersistentVolumeAccessMode{
						core.ReadWriteOnce,
					},
					Resources: core.ResourceRequirements{
						Requests: core.ResourceList{
							core.ResourceStorage: volumeSize,
						},
					},
				},
			},
		}
		if storageSettings.StorageClass != "" {
			statefulSet.Spec.VolumeClaimTemplates[0].Spec.StorageClassName = &storageSettings.StorageClass
		}
	}

	if instance.Spec.EnablePrometheus {
		templateSpec.Containers[0].Env = append(templateSpec.Containers[0].Env,
			core.EnvVar{Name: "NEO4J_metrics_prometheus_enabled", Value: "true"},
			core.EnvVar{Name: "NEO4J_metrics_prometheus_endpoint", Value: "localhost:2004"},
		)

		templateSpec.Containers[0].Ports = append(templateSpec.Containers[0].Ports,
			core.ContainerPort{Name: "tcp-prometheus", ContainerPort: 2004, Protocol: "TCP"},
		)
	}

	return statefulSet
}
