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
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type CoreService struct {
}

func (s *CoreService) Create(instance *neo4jv1alpha1.Neo4jCluster) (MetaObject, error) {
	return buildCoreService(instance), nil
}

func (s *CoreService) Update(instance *neo4jv1alpha1.Neo4jCluster, found runtime.Object) (MetaObject, bool, error) {
	other := found.(*core.Service)
	return other, false, nil
}

func (s *CoreService) GetName(instance *neo4jv1alpha1.Neo4jCluster) string {
	return instance.CoreServiceName()
}

func (s *CoreService) DefaultObject() runtime.Object {
	return &core.Service{}
}

type ReadReplicaService struct {
}

func (r *ReadReplicaService) Create(instance *neo4jv1alpha1.Neo4jCluster) (MetaObject, error) {
	if !instance.Spec.IsCausalCluster() {
		// Replicas are only meaningful if we have a core set.
		return nil, nil
	}
	return buildReplicaService(instance), nil
}

func (r *ReadReplicaService) Update(instance *neo4jv1alpha1.Neo4jCluster, found runtime.Object) (MetaObject, bool, error) {
	other := found.(*core.Service)
	if !instance.Spec.IsCausalCluster() {
		// Replicas are only meaningful if we have a core set.
		return nil, false, nil
	}
	return other, false, nil
}

func (r *ReadReplicaService) GetName(instance *neo4jv1alpha1.Neo4jCluster) string {
	return instance.ReadReplicaName()
}

func (r *ReadReplicaService) DefaultObject() runtime.Object {
	return &core.Service{}
}

type DiscoveryService struct {
	Index int
}

func (s *DiscoveryService) Create(instance *neo4jv1alpha1.Neo4jCluster) (MetaObject, error) {
	return buildDiscoveryService(s.Index, instance), nil
}

func (s *DiscoveryService) Update(instance *neo4jv1alpha1.Neo4jCluster, found runtime.Object) (MetaObject, bool, error) {
	other := found.(*core.Service)
	return other, false, nil
}

func (s *DiscoveryService) GetName(instance *neo4jv1alpha1.Neo4jCluster) string {
	return instance.DiscoveryServiceName(s.Index)
}

func (s *DiscoveryService) DefaultObject() runtime.Object {
	return &core.Service{}
}

func buildCoreService(instance *neo4jv1alpha1.Neo4jCluster) *core.Service {
	return &core.Service{
		ObjectMeta: meta.ObjectMeta{
			Name:      instance.CoreServiceName(),
			Namespace: instance.Namespace,
			Annotations: map[string]string{
				"service.alpha.kubernetes.io/tolerate-unready-endpoints": "true",
			},
			Labels: map[string]string{
				"neo4j.com/bolt": "true",
				"neo4j.com/http": "true",
			},
		},
		Spec: core.ServiceSpec{
			Ports: []core.ServicePort{
				{Name: "tcp-https", Port: 7473, TargetPort: intstr.FromInt(7473), Protocol: "TCP"},
				{Name: "tcp-http", Port: 7474, TargetPort: intstr.FromInt(7474), Protocol: "TCP"},
				{Name: "tcp-bolt", Port: 7687, TargetPort: intstr.FromInt(7687), Protocol: "TCP"},
				{Name: "tcp-backup", Port: 6362, TargetPort: intstr.FromInt(6362), Protocol: "TCP"},
			},
			Type:            "ClusterIP",
			SessionAffinity: "None",
			ClusterIP:       "None",
			Selector: map[string]string{
				"role":      "neo4j-core",
				"component": instance.LabelComponentName(),
			},
		},
	}
}

func buildReplicaService(instance *neo4jv1alpha1.Neo4jCluster) *core.Service {
	return &core.Service{
		ObjectMeta: meta.ObjectMeta{
			Name:      instance.ReadReplicaName(),
			Namespace: instance.Namespace,
			Labels: map[string]string{
				"neo4j.com/bolt": "true",
				"neo4j.com/http": "true",
			},
		},
		Spec: core.ServiceSpec{
			Ports: []core.ServicePort{
				{Name: "tcp-https", Port: 7473, TargetPort: intstr.FromInt(7473), Protocol: "TCP"},
				{Name: "tcp-http", Port: 7474, TargetPort: intstr.FromInt(7474), Protocol: "TCP"},
				{Name: "tcp-bolt", Port: 7687, TargetPort: intstr.FromInt(7687), Protocol: "TCP"},
			},
			Type:                     "ClusterIP",
			PublishNotReadyAddresses: true,
			SessionAffinity:          "None",
			ClusterIP:                "None",
			Selector: map[string]string{
				"role":      "neo4j-replica",
				"component": instance.LabelComponentName(),
			},
		},
	}
}

func buildDiscoveryService(idx int, instance *neo4jv1alpha1.Neo4jCluster) *core.Service {
	return &core.Service{
		ObjectMeta: meta.ObjectMeta{
			Name:      instance.DiscoveryServiceName(idx),
			Namespace: instance.Namespace,
			Labels: map[string]string{
				"neo4j.com/coreindex": fmt.Sprintf("%d", idx),
				"neo4j.com/cluster":   instance.LabelComponentName(),
				"neo4j.com/role":      "CORE",
				"neo4j.com/bolt":      "false",
				"neo4j.com/http":      "false",
			},
		},
		Spec: core.ServiceSpec{
			Ports: []core.ServicePort{
				{Name: "tcp-discovery", Port: 5000, TargetPort: intstr.FromInt(5000), Protocol: "TCP"},
				{Name: "tcp-transaction", Port: 6000, TargetPort: intstr.FromInt(6000), Protocol: "TCP"},
				{Name: "tcp-raft", Port: 7000, TargetPort: intstr.FromInt(7000), Protocol: "TCP"},
				{Name: "tcp-prometheus", Port: 2004, TargetPort: intstr.FromInt(2004), Protocol: "TCP"},
			},
			Type:                     "ClusterIP",
			PublishNotReadyAddresses: true,
			SessionAffinity:          "None",
			ClusterIP:                "None",
			Selector: map[string]string{
				"statefulset.kubernetes.io/pod-name": fmt.Sprintf("%s-%d", instance.CoreServiceName(), idx),
			},
		},
	}
}
