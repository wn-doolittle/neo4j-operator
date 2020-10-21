package reconciler

import (
	"fmt"
	neo4jv1alpha1 "github.com/wn-doolittle/neo4j-operator/api/v1alpha1"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type ServiceAccount struct {
}

func (s *ServiceAccount) Create(instance *neo4jv1alpha1.Neo4jCluster) (MetaObject, error) {
	sa := &core.ServiceAccount{
		ObjectMeta: meta.ObjectMeta{
			Name:      instance.ServiceAccountName(),
			Namespace: instance.Namespace,
			Labels: map[string]string{
				"component": instance.LabelComponentName(),
			},
		},
	}

	return sa, nil
}

func (s *ServiceAccount) Update(instance *neo4jv1alpha1.Neo4jCluster, found runtime.Object) (MetaObject, bool, error) {
	other := found.(*core.ServiceAccount)
	return other, false, nil
}

func (s *ServiceAccount) GetName(instance *neo4jv1alpha1.Neo4jCluster) string {
	return instance.ServiceAccountName()
}

func (s *ServiceAccount) DefaultObject() runtime.Object {
	return &core.ServiceAccount{}
}

type Role struct {
}

func (s *Role) Create(instance *neo4jv1alpha1.Neo4jCluster) (MetaObject, error) {
	r := &rbac.Role{
		ObjectMeta: meta.ObjectMeta{
			Name:      fmt.Sprintf("%s-reader", instance.ServiceAccountName()),
			Namespace: instance.Namespace,
			Labels: map[string]string{
				"component": instance.LabelComponentName(),
			},
		},
		Rules: []rbac.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"services"},
				Verbs:     []string{"get", "watch", "list"},
			},
		},
	}

	return r, nil
}

func (s *Role) Update(instance *neo4jv1alpha1.Neo4jCluster, found runtime.Object) (MetaObject, bool, error) {
	other := found.(*rbac.Role)
	return other, false, nil
}

func (s *Role) GetName(instance *neo4jv1alpha1.Neo4jCluster) string {
	return fmt.Sprintf("%s-reader", instance.ServiceAccountName())
}

func (s *Role) DefaultObject() runtime.Object {
	return &rbac.Role{}
}

type RoleBinding struct {
}

func (s *RoleBinding) Create(instance *neo4jv1alpha1.Neo4jCluster) (MetaObject, error) {
	r := &rbac.RoleBinding{
		ObjectMeta: meta.ObjectMeta{
			Name:      fmt.Sprintf("%s-reader-binding", instance.ServiceAccountName()),
			Namespace: instance.Namespace,
			Labels: map[string]string{
				"component": instance.LabelComponentName(),
			},
		},
		Subjects: []rbac.Subject{
			{
				Kind: "ServiceAccount",
				Name: instance.ServiceAccountName(),
			},
		},
		RoleRef: rbac.RoleRef{
			Kind:     "Role",
			Name:     fmt.Sprintf("%s-reader", instance.ServiceAccountName()),
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	return r, nil
}

func (s *RoleBinding) Update(instance *neo4jv1alpha1.Neo4jCluster, found runtime.Object) (MetaObject, bool, error) {
	other := found.(*rbac.RoleBinding)
	return other, false, nil
}

func (s *RoleBinding) GetName(instance *neo4jv1alpha1.Neo4jCluster) string {
	return fmt.Sprintf("%s-reader-binding", instance.ServiceAccountName())
}

func (s *RoleBinding) DefaultObject() runtime.Object {
	return &rbac.RoleBinding{}
}
