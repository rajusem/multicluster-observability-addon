package metrics_test

import (
	"encoding/json"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	prometheusv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	cooprometheusv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	cooprometheusv1alpha1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1alpha1"
	monitoringv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"
	clusterinfov1beta1 "github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	"github.com/stolostron/multicluster-observability-addon/internal/addon"
	"github.com/stolostron/multicluster-observability-addon/internal/addon/common"
	addoncfg "github.com/stolostron/multicluster-observability-addon/internal/addon/config"
	addonhelm "github.com/stolostron/multicluster-observability-addon/internal/addon/helm"
	"github.com/stolostron/multicluster-observability-addon/internal/analytics/rightsizing"
	"github.com/stolostron/multicluster-observability-addon/internal/metrics/config"
	internalres "github.com/stolostron/multicluster-observability-addon/internal/metrics/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	"open-cluster-management.io/addon-framework/pkg/addonfactory"
	"open-cluster-management.io/addon-framework/pkg/addonmanager/addontesting"
	"open-cluster-management.io/addon-framework/pkg/agent"
	"open-cluster-management.io/addon-framework/pkg/utils"
	addonapiv1beta1 "open-cluster-management.io/api/addon/v1beta1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	workv1 "open-cluster-management.io/api/work/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// mcoRightSizingMatch is the match[] list of the MCO-style right-sizing ScrapeConfig used below.
var mcoRightSizingMatch = []string{
	`{__name__="acm_rs:namespace:cpu_usage"}`,
	`{__name__="acm_rs_vm:namespace:cpu_usage"}`,
}

type objectIdentity struct {
	gk        schema.GroupKind
	namespace string
	name      string
}

// setupFullMCOA wires the full MCOA chart with the real values function. MCO's copy of the
// right-sizing ScrapeConfig reaches the addon through the configuration references, as it does
// on a hub where MCO deploys grafana/analytics/scrape-config.yaml.
func setupFullMCOA(t *testing.T, cooInstalled bool, rightSizing string) (agent.AgentAddon, *clusterv1.ManagedCluster, *addonapiv1beta1.ManagedClusterAddOn) {
	t.Helper()
	hubNamespace := "open-cluster-management-observability"

	scheme := runtime.NewScheme()
	require.NoError(t, kubescheme.AddToScheme(scheme))
	require.NoError(t, batchv1.AddToScheme(scheme))
	require.NoError(t, configv1.AddToScheme(scheme))
	require.NoError(t, cooprometheusv1alpha1.AddToScheme(scheme))
	require.NoError(t, monitoringv1alpha1.AddToScheme(scheme))
	require.NoError(t, prometheusv1.AddToScheme(scheme))
	require.NoError(t, cooprometheusv1.AddToScheme(scheme))
	require.NoError(t, clusterv1.Install(scheme))
	require.NoError(t, addonapiv1beta1.Install(scheme))
	require.NoError(t, workv1.Install(scheme))

	mcoCopy := &cooprometheusv1alpha1.ScrapeConfig{
		TypeMeta: metav1.TypeMeta{Kind: cooprometheusv1alpha1.ScrapeConfigsKind, APIVersion: cooprometheusv1alpha1.SchemeGroupVersion.Identifier()},
		ObjectMeta: metav1.ObjectMeta{
			Name:      rightsizing.ScrapeConfigName,
			Namespace: hubNamespace,
			Labels:    config.PlatformPrometheusMatchLabels,
		},
		Spec: cooprometheusv1alpha1.ScrapeConfigSpec{
			Params: map[string][]string{"match[]": mcoRightSizingMatch},
		},
	}
	platformSC := &cooprometheusv1alpha1.ScrapeConfig{
		TypeMeta: metav1.TypeMeta{Kind: cooprometheusv1alpha1.ScrapeConfigsKind, APIVersion: cooprometheusv1alpha1.SchemeGroupVersion.Identifier()},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "platform-metrics-default",
			Namespace: hubNamespace,
			Labels:    config.PlatformPrometheusMatchLabels,
		},
	}
	clusterVersion := &configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "version"},
		Spec:       configv1.ClusterVersionSpec{ClusterID: configv1.ClusterID(testClusterID)},
	}

	aodc := newAddonDeploymentConfig()
	aodc.Spec.CustomizedVariables = append(aodc.Spec.CustomizedVariables,
		addonapiv1beta1.CustomizedVariable{Name: addon.KeyPlatformMetricsCollection, Value: string(addon.PrometheusAgentV1alpha1)},
		addonapiv1beta1.CustomizedVariable{Name: addon.KeyMetricsHubHostname, Value: "metrics.example.com"},
		addonapiv1beta1.CustomizedVariable{Name: addon.KeyRightSizingDelegated, Value: "true"},
		addonapiv1beta1.CustomizedVariable{Name: addon.KeyPlatformNamespaceRightSizing, Value: rightSizing},
		addonapiv1beta1.CustomizedVariable{Name: addon.KeyPlatformVirtualizationRightSizing, Value: rightSizing},
	)

	managedCluster := addontesting.NewManagedCluster("cluster-1")
	managedCluster.Labels = map[string]string{
		addoncfg.ManagedClusterLabelClusterID: testClusterID,
		clusterinfov1beta1.LabelKubeVendor:    string(clusterinfov1beta1.KubeVendorOpenShift),
	}
	cmao := newCMOA()

	clientObjects := []client.Object{
		mcoCopy, platformSC, clusterVersion, aodc, managedCluster, cmao,
		newSecret(config.HubCASecretName, hubNamespace),
		newSecret(config.ClientCertSecretName, hubNamespace),
		newSecret(config.AlertmanagerAccessorSecretName, hubNamespace),
		newManifestWork("cluster-1", cooInstalled),
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: config.ImagesConfigMapObjKey.Name, Namespace: config.ImagesConfigMapObjKey.Namespace},
			Data: map[string]string{
				"obo_prometheus_rhel9_operator": "quay.io/prometheus/obo-operator",
				"prometheus_config_reloader":    "quay.io/prometheus/config-reloader",
				"kube_rbac_proxy":               "quay.io/kube/rbac-proxy",
				"kube_state_metrics":            "quay.io/kube/kube-state-metrics",
				"node_exporter":                 "quay.io/kube/node-exporter",
				"prometheus":                    "quay.io/prometheus/prometheus",
				"endpoint_monitoring_operator":  "quay.io/stolostron/endpoint-monitoring-operator",
			},
		},
	}
	configReferences := []addonapiv1beta1.ConfigReference{
		newConfigReference(mcoCopy), newConfigReference(platformSC), newConfigReference(aodc),
	}

	c := fakeclient.NewClientBuilder().
		WithInterceptorFuncs(ensureGVKIsSet(scheme)).
		WithScheme(scheme).
		WithObjects(clientObjects...).
		Build()

	defaultStack := internalres.DefaultStackResources{
		Client:       c,
		CMAO:         cmao,
		AddonOptions: newAddonOptions(true, false),
		Logger:       klog.Background(),
	}
	dc, err := defaultStack.Reconcile(t.Context())
	require.NoError(t, err)
	require.NoError(t, common.EnsureAddonConfig(t.Context(), klog.Background(), c, dc))

	agents := cooprometheusv1alpha1.PrometheusAgentList{}
	require.NoError(t, c.List(t.Context(), &agents))
	for i := range agents.Items {
		configReferences = append(configReferences, newConfigReference(&agents.Items[i]))
	}

	mca := addontesting.NewAddon("test", "cluster-1")
	mca.Status.ConfigReferences = configReferences

	getter := mockAODCGetter{aodc}
	agentAddon, err := addonfactory.NewAgentAddonFactory(addoncfg.Name, addon.FS, addoncfg.McoaChartDir).
		WithGetValuesFuncs(addonhelm.GetValuesFunc(t.Context(), c, getter, klog.Background())).
		WithAgentRegistrationOption(&agent.RegistrationOption{}).
		WithAgentInstallNamespace(utils.AgentInstallNamespaceFromDeploymentConfigFunc(getter)).
		WithScheme(scheme).
		BuildHelmAgentAddon()
	require.NoError(t, err)

	return agentAddon, managedCluster, mca
}

func identityOf(t *testing.T, obj runtime.Object) objectIdentity {
	t.Helper()
	acc, err := meta.Accessor(obj)
	require.NoError(t, err)
	return objectIdentity{gk: obj.GetObjectKind().GroupVersionKind().GroupKind(), namespace: acc.GetNamespace(), name: acc.GetName()}
}

// lastObjectPerIdentity mimics the ManifestWork builder, which keeps the last object it sees for
// each kind, namespace and name.
func lastObjectPerIdentity(t *testing.T, objects []runtime.Object) map[objectIdentity]string {
	t.Helper()
	ret := map[objectIdentity]string{}
	for _, obj := range objects {
		content, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		require.NoError(t, err)
		raw, err := json.Marshal(content)
		require.NoError(t, err)
		ret[identityOf(t, obj)] = string(raw)
	}
	return ret
}

func TestHelmBuild_MCOA_RightSizingScrapeConfig(t *testing.T) {
	t.Setenv("UNIT_TEST", "true")

	for _, tc := range []struct {
		name             string
		cooInstalled     bool
		rightSizing      string
		wantControllerID string
		wantManagedBy    string
	}{
		{
			name:             "right-sizing enabled with the MCOA prometheus operator",
			rightSizing:      "enabled",
			wantControllerID: config.PrometheusControllerID,
			wantManagedBy:    "multicluster-observability-addon-manager",
		},
		{
			name:             "right-sizing enabled with COO installed",
			cooInstalled:     true,
			rightSizing:      "enabled",
			wantControllerID: "",
			wantManagedBy:    "observability-operator",
		},
		{
			// MCO ships its copy whenever MCOA platform metrics are enabled.
			name:             "right-sizing disabled",
			rightSizing:      "disabled",
			wantControllerID: config.PrometheusControllerID,
			wantManagedBy:    "multicluster-observability-addon-manager",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			agentAddon, cluster, mca := setupFullMCOA(t, tc.cooInstalled, tc.rightSizing)

			// Raw helm output: every object must be rendered once.
			objects, err := agentAddon.Manifests(t.Context(), cluster, mca)
			require.NoError(t, err)

			seen := map[objectIdentity]int{}
			var rs []*cooprometheusv1alpha1.ScrapeConfig
			for _, obj := range objects {
				seen[identityOf(t, obj)]++
				if sc, ok := obj.(*cooprometheusv1alpha1.ScrapeConfig); ok && sc.Name == rightsizing.ScrapeConfigName {
					rs = append(rs, sc)
				}
			}
			for id, n := range seen {
				assert.Equal(t, 1, n, "object rendered more than once: %+v", id)
			}

			// MCO's copy is the only right-sizing ScrapeConfig, selected by the platform agent.
			require.Len(t, rs, 1)
			assert.Equal(t, config.PlatformPrometheusMatchLabels[addoncfg.ComponentK8sLabelKey], rs[0].Labels[addoncfg.ComponentK8sLabelKey])
			assert.Equal(t, tc.wantManagedBy, rs[0].Labels["app.kubernetes.io/managed-by"])
			assert.Equal(t, tc.wantControllerID, rs[0].Annotations["operator.prometheus.io/controller-id"])
			assert.ElementsMatch(t, mcoRightSizingMatch, rs[0].Spec.Params["match[]"])

			// Every render must produce the same ManifestWork content.
			want := lastObjectPerIdentity(t, objects)
			for range 10 {
				again, err := agentAddon.Manifests(t.Context(), cluster, mca)
				require.NoError(t, err)
				assert.Equal(t, want, lastObjectPerIdentity(t, again))
			}
		})
	}
}
