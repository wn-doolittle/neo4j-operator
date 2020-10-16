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
	neo4jv1alpha1 "github.com/wn-doolittle/neo4j-operator/api/v1alpha1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type MetaObject interface {
	meta.Object
	runtime.Object
}

type ManagedObject interface {
	Create(instance *neo4jv1alpha1.Neo4jCluster) (MetaObject, error)
	Update(instance *neo4jv1alpha1.Neo4jCluster, found runtime.Object) (MetaObject, bool, error)
	GetName(instance *neo4jv1alpha1.Neo4jCluster) string
	DefaultObject() runtime.Object
}
