package clusterinfo

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"reflect"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// Output of the probe script on node-1 (GMKtec M5 Pro).
const probeOutput = `cpu_model=AMD Ryzen 7 5700U with Radeon Graphics
cpu_model_fallback=
cpu_threads=16
cpu_cores=8
cpu_sockets=1
cpu_max_khz=4373858
dmi_sys_vendor=GMKtec
dmi_product_name=M5 Pro
dmi_product_version=Version 1.0
dmi_board_name=GMKtec
dmi_bios_version=M5 Pro 1.03
mem_kb=65186172
virtualized=0
`

func TestParseProbe(t *testing.T) {
	h := parseProbe(probeOutput)
	want := &Hardware{
		CPUModel: "AMD Ryzen 7 5700U with Radeon Graphics", CPUCores: 8, CPUThreads: 16, CPUSockets: 1, CPUMaxMHz: 4373,
		Vendor: "GMKtec", Product: "M5 Pro", ProductVersion: "Version 1.0", Board: "GMKtec", BIOS: "M5 Pro 1.03",
		MemoryBytes: 65186172 * 1024,
	}
	if !reflect.DeepEqual(h, want) {
		t.Fatalf("got %+v", h)
	}
	// ARM: no "model name", no "cpu cores"; VM with BIOS placeholders; two sockets.
	arm := parseProbe(
		"cpu_model=\ncpu_model_fallback=Raspberry Pi 4 Model B Rev 1.4\ncpu_threads=4\ncpu_sockets=0\n" +
			"dmi_sys_vendor=To Be Filled By O.E.M.\nvirtualized=1\n",
	)
	if arm.CPUModel != "Raspberry Pi 4 Model B Rev 1.4" || arm.CPUCores != 4 || arm.CPUSockets != 1 ||
		arm.Vendor != "" ||
		!arm.Virtualized {
		t.Errorf("arm/vm: %+v", arm)
	}
	dual := parseProbe("cpu_model=Xeon\ncpu_threads=32\ncpu_cores=8\ncpu_sockets=2\n")
	if dual.CPUCores != 16 {
		t.Errorf("cores must be per socket * sockets: %+v", dual)
	}
}

func TestRedactArgs(t *testing.T) {
	got := redactArgs(
		[]string{"oidc-login", "get-token", "--oidc-issuer-url=https://id.example.com", "--oidc-client-id=k8s",
			"--oidc-client-secret=s3cr3t", "--token", "abc", "--password=pw", "-v"},
	)
	want := []string{"oidc-login", "get-token", "--oidc-issuer-url=https://id.example.com", "--oidc-client-id=k8s",
		"--oidc-client-secret=••••", "--token", "••••", "--password=••••", "-v"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q", got)
	}
}

func selfSignedPEM(t *testing.T, cn string) []byte {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: cn, Organization: []string{"system:masters"}},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func TestSummarizeKubeconfig(t *testing.T) {
	cfg := clientcmdapi.Config{
		CurrentContext: "oidc",
		Contexts: map[string]*clientcmdapi.Context{
			"oidc":  {Cluster: "homelab", AuthInfo: "oidc-user"},
			"admin": {Cluster: "homelab", AuthInfo: "admin", Namespace: "kube-system"},
			"ci":    {Cluster: "homelab", AuthInfo: "ci"},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"oidc-user": {
				Exec: &clientcmdapi.ExecConfig{
					Command:    "kubectl",
					Args:       []string{"oidc-login", "get-token", "--oidc-client-secret=x"},
					APIVersion: "client.authentication.k8s.io/v1",
				},
			},
			"admin": {ClientCertificateData: selfSignedPEM(t, "admin")},
			"ci":    {Token: "eyJhbGciOiJSUzI1NiJ9.payload.signature"},
		},
	}
	kc, auth := summarizeKubeconfig(cfg, "", []string{"/home/u/.kube/config"})
	if kc.Context != "oidc" || kc.Cluster != "homelab" || kc.User != "oidc-user" || len(kc.Contexts) != 3 {
		t.Errorf("kubeconfig: %+v", kc)
	}
	if auth.Method != "exec" || auth.ExecCommand != "kubectl" || auth.ExecArgs[2] != "--oidc-client-secret=••••" {
		t.Errorf("exec auth: %+v", auth)
	}
	kc, auth = summarizeKubeconfig(cfg, "admin", nil)
	if kc.Context != "admin" || kc.Namespace != "kube-system" || auth.Method != "client-certificate" ||
		auth.ClientCertificate == nil ||
		!strings.Contains(auth.ClientCertificate.Subject, "CN=admin") ||
		!strings.Contains(auth.ClientCertificate.Subject, "system:masters") {
		t.Errorf("cert auth: %+v %+v", kc, auth.ClientCertificate)
	}
	_, auth = summarizeKubeconfig(cfg, "ci", nil)
	if auth.Method != "token" || strings.Contains(auth.TokenPreview, "payload") || auth.TokenPreview != "eyJh…re" {
		t.Errorf("token must be masked: %+v", auth)
	}
	_, auth = summarizeKubeconfig(cfg, "missing", nil)
	if auth.Method != "none" {
		t.Errorf("unknown context: %+v", auth)
	}
}

func TestBuildNodeAndTotals(t *testing.T) {
	q := resource.MustParse
	n := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "m1",
			Labels: map[string]string{"node-role.kubernetes.io/control-plane": ""},
		},
		Spec: corev1.NodeSpec{
			Taints: []corev1.Taint{
				{Key: "node-role.kubernetes.io/control-plane", Effect: corev1.TaintEffectNoSchedule},
			},
		},
		Status: corev1.NodeStatus{
			Capacity: corev1.ResourceList{
				corev1.ResourceCPU:    q("16"),
				corev1.ResourceMemory: q("64Gi"),
				corev1.ResourcePods:   q("110"),
			},
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				{Type: corev1.NodeMemoryPressure, Status: corev1.ConditionTrue},
			},
			Addresses: []corev1.NodeAddress{{Type: corev1.NodeInternalIP, Address: "10.0.0.1"}},
		},
	}
	node := buildNode(n)
	if !node.Ready || node.Roles[0] != "control-plane" || node.Pressure[0] != "MemoryPressure" ||
		node.InternalIP != "10.0.0.1" ||
		node.Taints[0] != "node-role.kubernetes.io/control-plane:NoSchedule" ||
		node.CPUCapacityMillis != 16000 {
		t.Errorf("node: %+v", node)
	}
	worker := buildNode(&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "w1"}})
	if worker.Roles[0] != "worker" || worker.Ready || worker.Status != "Unknown" {
		t.Errorf("worker: %+v", worker)
	}
	hw := &Hardware{CPUModel: "Ryzen", CPUCores: 8}
	a, b := node, node
	a.Hardware, b.Hardware = hw, hw
	a.Usage = &Usage{CPUMillis: 500, MemoryBytes: 1 << 30}
	tt := totals([]Node{a, b, worker}, true)
	if tt.Nodes != 3 || tt.ReadyNodes != 2 || tt.CPUCapacityMillis != 32000 || tt.PhysicalCores != 16 ||
		tt.CPUUsedMillis != 500 ||
		len(tt.CPUModels) != 1 ||
		tt.CPUModels[0].Count != 2 {
		t.Errorf("totals: %+v", tt)
	}
}
