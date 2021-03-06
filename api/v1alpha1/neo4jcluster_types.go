/*


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

package v1alpha1

import (
	"encoding/base64"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// Neo4jClusterSpec defines the desired state of Neo4jCluster
type Neo4jClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	ImageVersion         string             `json:"image-version"`
	ImagePullPolicy      string             `json:"image-pull-policy,omitempty"`
	AdminPassword        string             `json:"admin-password,omitempty"`
	CoreServers          int32              `json:"core-replicas"`
	CoreArguments        map[string]string  `json:"core-args,omitempty"`
	ReadReplicaServers   int32              `json:"read-replica-replicas"`
	ReadReplicaArguments map[string]string  `json:"read-replica-args,omitempty"`
	Resources            Resources          `json:"resources"`
	PodAnnotations       map[string]string  `json:"pod-annotations,omitempty"`
	PersistentStorage    *PersistentStorage `json:"persistent-storage,omitempty"`
	SslCertificates      *SslCertificates   `json:"ssl,omitempty"`
	Backup               *Backup            `json:"backup,omitempty"`
	NodeSelector         map[string]string  `json:"node-selector,omitempty"`
	EnablePrometheus     bool               `json:"enable-prometheus,omitempty"`
}

type PersistentStorage struct {
	Size         string `json:"size"`
	StorageClass string `json:"storage-class,omitempty"`
	MountPath    string `json:"mount-path,omitempty"`
}

type SslCertificates struct {
	PrivateKey        string `json:"key"`
	PublicCertificate string `json:"certificate"`
}

type Backup struct {
	Schedule     string    `json:"schedule"`
	Size         string    `json:"size"`
	StorageClass string    `json:"storage-class,omitempty"`
	Resources    Resources `json:"resources"`
}

type Resources struct {
	Requests MemoryCPU `json:"requests"`
	Limits   MemoryCPU `json:"limits"`
}

type MemoryCPU struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

// Neo4jClusterStatus defines the observed state of Neo4jCluster
type Neo4jClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	CoreStats    string `json:"core-stats,omitempty"`
	ReplicaStats string `json:"replica-stats,omitempty"`
	Leader       string `json:"leader,omitempty"`
	BoltURL      string `json:"bolt-url,omitempty"`
	State        string `json:"state,omitempty"`
	Message      string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Neo4jCluster is the Schema for the neo4jclusters API
type Neo4jCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   Neo4jClusterSpec   `json:"spec,omitempty"`
	Status Neo4jClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// Neo4jClusterList contains a list of Neo4jCluster
type Neo4jClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Neo4jCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Neo4jCluster{}, &Neo4jClusterList{})
}

// Custom functions.
func (i *Neo4jCluster) ServiceAccountName() string {
	return fmt.Sprintf("neo4j-%s-sa", i.Name)
}

func (i *Neo4jCluster) SecretStoreName() string {
	return fmt.Sprintf("neo4j-%s-secrets", i.Name)
}

func (i *Neo4jCluster) CommonConfigMapName() string {
	return fmt.Sprintf("neo4j-%s-common-config", i.Name)
}

func (i *Neo4jCluster) CoreConfigMapName() string {
	return fmt.Sprintf("neo4j-%s-core-config", i.Name)
}

func (i *Neo4jCluster) ReplicaConfigMapName() string {
	return fmt.Sprintf("neo4j-%s-replica-config", i.Name)
}

func (i *Neo4jCluster) CoreServiceName() string {
	return fmt.Sprintf("neo4j-core-%s", i.Name)
}

func (i *Neo4jCluster) RandomCorePod() string {
	// return fmt.Sprintf("neo4j-core-%s-%d.%s", i.Name, rand.Intn(int(i.Spec.CoreServers)), i.CoreServiceName())
	// For now let us always return he first pod so that shrinking
	// and expanding the cluster does not influence backup jobs.
	return fmt.Sprintf("neo4j-core-%s-%d.%s", i.Name, 0, i.CoreServiceName())
}

func (i *Neo4jCluster) DiscoveryServiceNamePrefix() string {
	return fmt.Sprintf("discovery-neo4j-%s", i.Name)
}

func (i *Neo4jCluster) DiscoveryServiceName(idx int) string {
	return fmt.Sprintf("%s-%d", i.DiscoveryServiceNamePrefix(), idx)
}

func (i *Neo4jCluster) ReadReplicaName() string {
	return fmt.Sprintf("neo4j-replica-%s", i.Name)
}

func (i *Neo4jCluster) LabelComponentName() string {
	return fmt.Sprintf("neo4j-%s", i.Name)
}

func (i *Neo4jClusterSpec) DockerImage() string {
	return fmt.Sprintf("neo4j:%s", i.ImageVersion)
}

func (i *Neo4jClusterSpec) IsCausalCluster() bool {
	return i.CoreServers >= 3
}

func (i *Neo4jClusterSpec) AuthorizationEnabled() bool {
	return i.AdminPassword != ""
}

func (i *Neo4jClusterSpec) AdminPasswordClearText() (*string, error) {
	if i.AuthorizationEnabled() {
		pwd, err := base64.StdEncoding.DecodeString(i.AdminPassword)
		if err != nil {
			return nil, err
		}
		p := string(pwd)
		return &p, nil
	}
	return nil, nil
}
