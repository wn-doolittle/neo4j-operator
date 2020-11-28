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

package controllers

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	neo4jv1alpha1 "github.com/wn-doolittle/neo4j-operator/api/v1alpha1"
	"github.com/wn-doolittle/neo4j-operator/controllers/backup"
	"github.com/wn-doolittle/neo4j-operator/controllers/reconciler"
)

const ReconcileTime = 30 * time.Second

// Neo4jClusterReconciler reconciles a Neo4jCluster object
type Neo4jClusterReconciler struct {
	Client    client.Client
	ClientSet *kubernetes.Clientset
	Log       logr.Logger
	Scheme    *runtime.Scheme
}

var _ reconcile.Reconciler = &Neo4jClusterReconciler{}

var managedObjects = []reconciler.ManagedObject{
	&reconciler.ServiceAccount{},
	&reconciler.Role{},
	&reconciler.RoleBinding{},
	&reconciler.CommonConfigMap{},
	&reconciler.CoreConfigMap{},
	&reconciler.ReplicaConfigMap{},
	&reconciler.InitScriptConfigMap{},
	&reconciler.Secret{},
	&reconciler.CoreServer{},
	&reconciler.CoreService{},
	&reconciler.DiscoveryService{Index: 0},
	&reconciler.DiscoveryService{Index: 1},
	&reconciler.DiscoveryService{Index: 2},
	&reconciler.DiscoveryService{Index: 3},
	&reconciler.DiscoveryService{Index: 4},
	&reconciler.DiscoveryService{Index: 5},
	&reconciler.ReadReplica{},
	&reconciler.ReadReplicaService{},
}

// +kubebuilder:rbac:groups=neo4j.database.wna.cloud,resources=neo4jclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=neo4j.database.wna.cloud,resources=neo4jclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=pods;services;endpoints;persistentvolumeclaims;events;configmaps;secrets;serviceaccounts,verbs=*
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=*
// +kubebuilder:rbac:groups=apps,resources=deployments;daemonsets;replicasets;statefulsets,verbs=*
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors,verbs=get;create
// +kubebuilder:rbac:groups=apps,resourceNames=neo4j-operator,resources=deployments/finalizers,verbs=update

func (r *Neo4jClusterReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	reqLogger := r.Log.WithValues("neo4jcluster", request.NamespacedName)

	// reqLogger.Info("Reconciling Neo4jCluster started...")

	// Fetch the Neo4jCluster instance
	instance := &neo4jv1alpha1.Neo4jCluster{}
	err := r.Client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	err = backup.ScheduleBackup(r.ClientSet, reqLogger, instance)
	if err != nil {
		reqLogger.Error(err, "Failed to schedule backup")
		return reconcile.Result{}, err
	}

	// TODO(lantonia): Build external service with ports (https, bolt) configurable. Can we really support internal and external access at once?
	for _, obj := range managedObjects {
		found := obj.DefaultObject()
		err := r.Client.Get(context.TODO(), types.NamespacedName{Name: obj.GetName(instance), Namespace: request.Namespace}, found)
		if err != nil && errors.IsNotFound(err) {
			result, err := obj.Create(instance)
			if err != nil {
				return reconcile.Result{}, err
			}
			if result != nil {
				reqLogger.Info(fmt.Sprintf("Creating new %T", result), "Namespace", result.GetNamespace(), "Name", result.GetName())
				err = r.Client.Create(context.TODO(), result)
				if err != nil {
					reqLogger.Error(err, "Object creation failed", "Namespace", result.GetNamespace(), "Name", result.GetName())
					return reconcile.Result{}, err
				}
			}
		} else if err != nil {
			return reconcile.Result{}, err
		} else {
			result, restart, err := obj.Update(instance, found.DeepCopyObject())
			if err != nil {
				return reconcile.Result{}, err
			}
			if result == nil {
				reqLogger.Info(fmt.Sprintf("Deleting %T", found), "Namespace", request.Namespace, "Name", obj.GetName(instance))
				err = r.Client.Delete(context.TODO(), found)
				if err != nil {
					reqLogger.Error(err, "Object deletion failed", "Namespace", request.Namespace, "Name", obj.GetName(instance))
					return reconcile.Result{}, err
				}
			} else if !reflect.DeepEqual(result, found) {
				reqLogger.Info(fmt.Sprintf("Updating existing %T", result), "Namespace", result.GetNamespace(), "Name", result.GetName())
				err = r.Client.Update(context.TODO(), result)
				if err != nil {
					reqLogger.Error(err, "Object update failed", "Namespace", result.GetNamespace(), "Name", result.GetName())
					return reconcile.Result{}, err
				}
				if restart {
					// We use rolling upgrade strategy, so there is no need to manually restart pods.
					//err = rollingPodRestart(reqLogger, r, instance)
					//if err != nil {
					//	reqLogger.Error(err, "Cluster rolling restart failed", "Namespace", result.GetNamespace(), "Name", result.GetName())
					//	return reconcile.Result{}, err
					//}
				}
			}
		}
	}

	err = r.updateClusterStatus(instance)
	if err != nil {
		reqLogger.Error(err, "Failed to update cluster status")
		return reconcile.Result{}, err
	}

	// reqLogger.Info("Reconciliation of Neo4jCluster completed.")
	return reconcile.Result{RequeueAfter: ReconcileTime}, nil
}

func rollingPodRestart(logger logr.Logger, r *Neo4jClusterReconciler, instance *neo4jv1alpha1.Neo4jCluster) error {
	foundPods := &core.PodList{}
	err := r.Client.List(context.TODO(), foundPods,
		client.InNamespace(instance.Namespace), client.MatchingLabels(map[string]string{
			"component": instance.LabelComponentName(),
		}))
	if err != nil {
		return err
	}
	for _, p := range foundPods.Items {
		err = restartPod(logger, r, instance, p)
		if err != nil {
			return err
		}
	}
	return nil
}

func restartPod(logger logr.Logger, r *Neo4jClusterReconciler, instance *neo4jv1alpha1.Neo4jCluster, pod core.Pod) error {
	logger.Info(fmt.Sprintf("Restarting pod %s.", pod.Name))
	err := r.ClientSet.CoreV1().Pods(instance.Namespace).Delete(context.TODO(), pod.Name, meta.DeleteOptions{})
	if err != nil {
		return err
	}
	for {
		refreshed, err := r.ClientSet.CoreV1().Pods(instance.Namespace).Get(context.TODO(), pod.Name, meta.GetOptions{})
		if err != nil {
			return err
		}
		if refreshed.DeletionTimestamp == nil {
			ready := true
			for _, c := range refreshed.Status.ContainerStatuses {
				if !c.Ready {
					ready = false
				}
			}
			if ready {
				return nil
			}
		} else {
			// Pod is being removed, wait.
		}
		logger.Info(fmt.Sprintf("Waiting for pod %s to become ready...", refreshed.Name))
		time.Sleep(10 * time.Second)
	}
}

func (r *Neo4jClusterReconciler) updateClusterStatus(instance *neo4jv1alpha1.Neo4jCluster) error {
	cOn, cOff, err := findOnlineOfflinePods(r, client.InNamespace(instance.Namespace), client.MatchingLabels(map[string]string{
		"component": instance.LabelComponentName(),
		"role":      "neo4j-core",
	}))
	if err != nil {
		return err
	}
	rOn, rOff, err := findOnlineOfflinePods(r, client.InNamespace(instance.Namespace), client.MatchingLabels(map[string]string{
		"component": instance.LabelComponentName(),
		"role":      "neo4j-replica",
	}))
	if err != nil {
		return err
	}
	if !instance.Spec.IsCausalCluster() {
		instance.Status.Leader = "N/A"
	} else {
		instance.Status.Leader = discoverLeader(instance, cOn)
	}
	instance.Status.CoreStats = fmt.Sprintf("%d/%d", len(cOn), len(cOn)+len(cOff))
	instance.Status.ReplicaStats = fmt.Sprintf("%d/%d", len(rOn), len(rOn)+len(rOff))
	if !instance.Spec.IsCausalCluster() {
		instance.Status.BoltURL = fmt.Sprintf("bolt://neo4j-core-%s:7687", instance.Name)
	} else {
		instance.Status.BoltURL = fmt.Sprintf("bolt+routing://neo4j-core-%s:7687", instance.Name)
	}
	instance.Status.State = ""
	instance.Status.Message = ""
	return r.Client.Status().Update(context.TODO(), instance)
}

// TODO(lantonia): Refactor methods so that you do not pass Neo4JCluster object everywhere.
func discoverLeader(instance *neo4jv1alpha1.Neo4jCluster, corePods []string) string {
	leader := ""
	adminPassword, _ := instance.Spec.AdminPasswordClearText()
	ch := make(chan string)
	for _, coreServer := range corePods {
		go func(ch chan string, instance string, password *string) {
			httpClient := http.Client{Timeout: 5 * time.Second}
			req, _ := http.NewRequest("GET", fmt.Sprintf("http://%s:7474/db/manage/server/core/writable", instance), nil)
			if password != nil {
				req.SetBasicAuth("neo4j", *password)
			}
			res, err := httpClient.Do(req)
			if err == nil {
				defer res.Body.Close()
			}
			if err != nil || res.StatusCode != http.StatusOK {
				ch <- ""
			} else {
				body, _ := ioutil.ReadAll(res.Body)
				if string(body) != "true" {
					ch <- ""
				} else {
					ch <- instance
				}
			}
		}(ch, fmt.Sprintf("%s.%s", coreServer, instance.CoreServiceName()), adminPassword)
	}
	for range corePods {
		if response := <-ch; response != "" {
			leader = response
		}
	}
	return leader
}

func findOnlineOfflinePods(r *Neo4jClusterReconciler, listOpts ...client.ListOption) ([]string, []string, error) {
	foundPods := &core.PodList{}
	err := r.Client.List(context.TODO(), foundPods, listOpts...)
	if err != nil {
		return nil, nil, err
	}
	var (
		readyMembers   []string
		unreadyMembers []string
	)
	for _, p := range foundPods.Items {
		ready := true
		for _, c := range p.Status.ContainerStatuses {
			if !c.Ready {
				ready = false
			}
		}
		if ready {
			readyMembers = append(readyMembers, p.Name)
		} else {
			unreadyMembers = append(unreadyMembers, p.Name)
		}
	}
	return readyMembers, unreadyMembers, nil
}

func (r *Neo4jClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&neo4jv1alpha1.Neo4jCluster{}).
		Owns(&apps.StatefulSet{}).
		Owns(&core.Service{}).
		Owns(&core.Pod{}).
		Owns(&core.Secret{}).
		Complete(r)
}
