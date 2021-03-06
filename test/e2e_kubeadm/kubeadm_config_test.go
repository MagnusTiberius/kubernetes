/*
Copyright 2019 The Kubernetes Authors.

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

package e2e_kubeadm

import (
	yaml "gopkg.in/yaml.v2"
	authv1 "k8s.io/api/authorization/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	e2elog "k8s.io/kubernetes/test/e2e/framework/log"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	kubeadmConfigName                             = "kubeadm-config"
	kubeadmConfigRoleName                         = "kubeadm:nodes-kubeadm-config"
	kubeadmConfigRoleBindingName                  = kubeadmConfigRoleName
	kubeadmConfigClusterConfigurationConfigMapKey = "ClusterConfiguration"
	kubeadmConfigClusterStatusConfigMapKey        = "ClusterStatus"
)

var (
	kubeadmConfigConfigMapResource = &authv1.ResourceAttributes{
		Namespace: kubeSystemNamespace,
		Name:      kubeadmConfigName,
		Resource:  "configmaps",
		Verb:      "get",
	}
)

// Define container for all the test specification aimed at verifying
// that kubeadm creates the cluster-info ConfigMap, that it is properly configured
// and that all the related RBAC rules are in place
var _ = KubeadmDescribe("kubeadm-config ConfigMap", func() {

	// Get an instance of the k8s test framework
	f := framework.NewDefaultFramework("kubeadm-config")

	// Tests in this container are not expected to create new objects in the cluster
	// so we are disabling the creation of a namespace in order to get a faster execution
	f.SkipNamespaceCreation = true

	It("should exist and be properly configured", func() {
		cm := GetConfigMap(f.ClientSet, kubeSystemNamespace, kubeadmConfigName)

		Expect(cm.Data).To(HaveKey(kubeadmConfigClusterConfigurationConfigMapKey))
		Expect(cm.Data).To(HaveKey(kubeadmConfigClusterStatusConfigMapKey))

		m := unmarshalYaml(cm.Data[kubeadmConfigClusterStatusConfigMapKey])
		if _, ok := m["apiEndpoints"]; ok {
			d := m["apiEndpoints"].(map[interface{}]interface{})
			// get all control-plane nodes
			controlPlanes := getControlPlaneNodes(f.ClientSet)

			// checks that all the control-plane nodes are in the apiEndpoints list
			for _, cp := range controlPlanes.Items {
				if _, ok := d[cp.Name]; !ok {
					e2elog.Failf("failed to get apiEndpoints for control-plane %s in %s", cp.Name, kubeadmConfigClusterStatusConfigMapKey)
				}
			}
		} else {
			e2elog.Failf("failed to get apiEndpoints from %s", kubeadmConfigClusterStatusConfigMapKey)
		}
	})

	It("should have related Role and RoleBinding", func() {
		ExpectRole(f.ClientSet, kubeSystemNamespace, kubeadmConfigRoleName)
		ExpectRoleBinding(f.ClientSet, kubeSystemNamespace, kubeadmConfigRoleBindingName)
	})

	It("should be accessible for bootstrap tokens", func() {
		ExpectSubjectHasAccessToResource(f.ClientSet,
			rbacv1.GroupKind, bootstrapTokensGroup,
			kubeadmConfigConfigMapResource,
		)
	})

	It("should be accessible for for nodes", func() {
		ExpectSubjectHasAccessToResource(f.ClientSet,
			rbacv1.GroupKind, nodesGroup,
			kubeadmConfigConfigMapResource,
		)
	})
})

func getClusterConfiguration(c clientset.Interface) map[interface{}]interface{} {
	cm := GetConfigMap(c, kubeSystemNamespace, kubeadmConfigName)

	Expect(cm.Data).To(HaveKey(kubeadmConfigClusterConfigurationConfigMapKey))

	return unmarshalYaml(cm.Data[kubeadmConfigClusterConfigurationConfigMapKey])
}

func unmarshalYaml(data string) map[interface{}]interface{} {
	m := make(map[interface{}]interface{})
	err := yaml.Unmarshal([]byte(data), &m)
	if err != nil {
		e2elog.Failf("error parsing %s ConfigMap: %v", kubeadmConfigName, err)
	}
	return m
}
