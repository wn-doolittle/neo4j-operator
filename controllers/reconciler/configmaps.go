package reconciler

import (
	neo4jv1alpha1 "github.com/wn-doolittle/neo4j-operator/api/v1alpha1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/wn-doolittle/neo4j-operator/controllers/reconciler/scripts"
)

type CommonConfigMap struct {
}

func (c *CommonConfigMap) Create(instance *neo4jv1alpha1.Neo4jCluster) (MetaObject, error) {
	var props = map[string]string{
		"NEO4J_ACCEPT_LICENSE_AGREEMENT": "yes",
		"NUMBER_OF_CORES":                "3",
		"AUTH_ENABLED":                   "true",
		// Enable listening for backups; default port 6362
		"NEO4J_dbms_backup_enabled":                                            "true",
		"NEO4J_dbms_backup_listen__address":                                    "0.0.0.0:6362",
		"NEO4J_dbms_default__database":                                         "neo4j",
		"NEO4J_dbms_connector_bolt_listen__address":                            "0.0.0.0:7687",
		"NEO4J_dbms_connector_http_listen__address":                            "0.0.0.0:7474",
		"NEO4J_dbms_connector_https_listen__address":                           "0.0.0.0:7473",
		"NEO4J_causal__clustering_minimum__core__cluster__size__at__formation": "3",
		"NEO4J_causal__clustering_minimum__core__cluster__size__at__runtime":   "2",
		"NEO4J_dbms_jvm_additional":                                            "-XX:+ExitOnOutOfMemoryError",
		"NEO4JLABS_PLUGINS":                                                    "[\"apoc\"]",
		"NEO4J_apoc_import_file_use__neo4j__config":                            "true",
	}

	cm := createConfigMap(instance.CommonConfigMapName(), instance, props)

	return cm, nil
}

func (s *CommonConfigMap) Update(instance *neo4jv1alpha1.Neo4jCluster, found runtime.Object) (MetaObject, bool, error) {
	other := found.(*core.ConfigMap)
	return other, false, nil
}

func (s *CommonConfigMap) GetName(instance *neo4jv1alpha1.Neo4jCluster) string {
	return instance.CommonConfigMapName()
}

func (s *CommonConfigMap) DefaultObject() runtime.Object {
	return &core.ConfigMap{}
}

type CoreConfigMap struct {
}

func (c *CoreConfigMap) Create(instance *neo4jv1alpha1.Neo4jCluster) (MetaObject, error) {
	var props = map[string]string{
		"NEO4J_dbms_directories_logs": "/data/logs",
		"NEO4J_dbms_mode":             "CORE",
	}

	cm := createConfigMap(instance.CoreConfigMapName(), instance, props)

	return cm, nil
}

func (s *CoreConfigMap) Update(instance *neo4jv1alpha1.Neo4jCluster, found runtime.Object) (MetaObject, bool, error) {
	other := found.(*core.ConfigMap)
	return other, false, nil
}

func (s *CoreConfigMap) GetName(instance *neo4jv1alpha1.Neo4jCluster) string {
	return instance.CoreConfigMapName()
}

func (s *CoreConfigMap) DefaultObject() runtime.Object {
	return &core.ConfigMap{}
}

type ReplicaConfigMap struct {
}

func (c *ReplicaConfigMap) Create(instance *neo4jv1alpha1.Neo4jCluster) (MetaObject, error) {
	var props = map[string]string{
		"NEO4J_dbms_directories_logs": "/data/logs",
		"NEO4J_dbms_mode":             "READ_REPLICA",
	}

	cm := createConfigMap(instance.ReplicaConfigMapName(), instance, props)

	return cm, nil
}

func (s *ReplicaConfigMap) Update(instance *neo4jv1alpha1.Neo4jCluster, found runtime.Object) (MetaObject, bool, error) {
	other := found.(*core.ConfigMap)
	return other, false, nil
}

func (s *ReplicaConfigMap) GetName(instance *neo4jv1alpha1.Neo4jCluster) string {
	return instance.ReplicaConfigMapName()
}

func (s *ReplicaConfigMap) DefaultObject() runtime.Object {
	return &core.ConfigMap{}
}

type InitScriptConfigMap struct {
}

func (c *InitScriptConfigMap) Create(instance *neo4jv1alpha1.Neo4jCluster) (MetaObject, error) {
	cm := createConfigMap("neo4j-init-script", instance, map[string]string{
		"init.sh": scripts.Neo4jInitScript,
	})

	return cm, nil
}

func (s *InitScriptConfigMap) Update(instance *neo4jv1alpha1.Neo4jCluster, found runtime.Object) (MetaObject, bool, error) {
	other := found.(*core.ConfigMap)
	return other, false, nil
}

func (s *InitScriptConfigMap) GetName(instance *neo4jv1alpha1.Neo4jCluster) string {
	return "neo4j-init-script"
}

func (s *InitScriptConfigMap) DefaultObject() runtime.Object {
	return &core.ConfigMap{}
}

func createConfigMap(name string, instance *neo4jv1alpha1.Neo4jCluster, data map[string]string) *core.ConfigMap {
	return &core.ConfigMap{
		ObjectMeta: meta.ObjectMeta{
			Name:      name,
			Namespace: instance.Namespace,
			Labels: map[string]string{
				"component": instance.LabelComponentName(),
			},
		},
		Data: data,
	}
}
