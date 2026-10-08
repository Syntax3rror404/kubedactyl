package gameserver

import (
	"slices"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"app/api/v1alpha1"
)

func testServer() (*v1alpha1.GameServer, *v1alpha1.Egg) {
	gs := &v1alpha1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "survival-abcde", Namespace: "kubedactyl", UID: "uid-1"},
		Spec: v1alpha1.GameServerSpec{
			EggRef: "paper", Image: "ghcr.io/pelican-eggs/yolks:java_21",
			Environment: map[string]string{"SERVER_JARFILE": "paper.jar", "SERVER_PORT": "1"},
			Resources:   v1alpha1.Resources{MemoryMiB: 2048, CPUMillis: 1500, DiskMiB: 10240},
			Ports:       []int32{25565, 25575}, StopTimeoutSeconds: 600,
		},
	}
	e := &v1alpha1.Egg{Spec: v1alpha1.EggSpec{
		Startup: "java -jar {{SERVER_JARFILE}}",
		StartupCommands: []v1alpha1.StartupCommand{
			{Name: "Default", Command: "java -jar {{SERVER_JARFILE}}"},
			{Name: "Flags", Command: "java -XX:+UseG1GC -jar {{SERVER_JARFILE}}"},
		},
		Variables: []v1alpha1.EggVariable{
			{EnvVariable: "SERVER_JARFILE", DefaultValue: "server.jar"},
			{EnvVariable: "BUILD_NUMBER", DefaultValue: "latest"},
			{EnvVariable: "SERVER_PORT", DefaultValue: "reserved"},
		},
	}}
	return gs, e
}

func TestEnvironment(t *testing.T) {
	gs, e := testServer()
	env := Environment(gs, e, Options{Timezone: "Europe/Berlin"})
	want := map[string]string{
		"SERVER_JARFILE": "paper.jar", // server value
		"BUILD_NUMBER":   "latest",    // egg default
		"SERVER_PORT":    "25565",     // system value wins over the variable
		"SERVER_MEMORY":  "2048",
		"SERVER_IP":      "0.0.0.0",
		"STARTUP":        "java -jar {{SERVER_JARFILE}}", // raw, substituted by the image entrypoint
		"TZ":             "Europe/Berlin",
		"P_SERVER_UUID":  "uid-1",
	}
	for k, v := range want {
		if env[k] != v {
			t.Errorf("%s = %q, want %q", k, env[k], v)
		}
	}
}

func TestStartupCommand(t *testing.T) {
	gs, e := testServer()
	for name, want := range map[string]string{
		"":        "java -jar {{SERVER_JARFILE}}",
		"Flags":   "java -XX:+UseG1GC -jar {{SERVER_JARFILE}}",
		"Removed": "java -jar {{SERVER_JARFILE}}", // the egg no longer has it: its default
	} {
		gs.Spec.StartupName = name
		if got := StartupCommand(gs, e); got != want {
			t.Errorf("%q: %q, want %q", name, got, want)
		}
	}
	gs.Spec.Startup = "./own"
	if got := StartupCommand(gs, e); got != "./own" {
		t.Errorf("the server's own command must win: %q", got)
	}
}

func TestResolvePlaceholders(t *testing.T) {
	gs, e := testServer()
	env := Environment(gs, e, Options{})
	cases := map[string]string{
		"{{server.build.default.port}}":           "25565",
		"{{server.allocations.default.port}}":     "25565",
		"{{server.build.memory}}":                 "2048",
		"{{server.build.env.SERVER_JARFILE}}":     "paper.jar",
		"{{server.environment.BUILD_NUMBER}}":     "latest",
		"{{env.SERVER_PORT}}":                     "25565",
		"0.0.0.0:{{ server.build.default.port }}": "0.0.0.0:25565",
		"{{config.docker.interface}}":             "{{config.docker.interface}}", // left for the parser
		"{{server.build.env.UNKNOWN}}":            "{{server.build.env.UNKNOWN}}",
	}
	for in, want := range cases {
		if got := ResolvePlaceholders(in, gs, env); got != want {
			t.Errorf("%s -> %q, want %q", in, got, want)
		}
	}
}

func TestGamePod(t *testing.T) {
	gs, e := testServer()
	pod := GamePod(gs, e, Options{})
	c := pod.Spec.Containers[0]
	if !c.TTY || !c.Stdin {
		t.Error("game container needs tty and stdin")
	}
	if *pod.Spec.SecurityContext.RunAsUser != 988 || *pod.Spec.SecurityContext.RunAsGroup != 988 {
		t.Error("game process must run as 988:988")
	}
	if !*c.SecurityContext.ReadOnlyRootFilesystem {
		t.Error("root filesystem must be read-only")
	}
	if *pod.Spec.EnableServiceLinks {
		t.Error("service links would pollute the environment")
	}
	// 2048 MiB * 1.15 overhead
	if mem := c.Resources.Limits[corev1.ResourceMemory]; mem.Value() != 2355<<20 {
		t.Errorf("memory limit = %s", mem.String())
	}
	if cpu := c.Resources.Limits[corev1.ResourceCPU]; cpu.MilliValue() != 1500 {
		t.Errorf("cpu limit = %s", cpu.String())
	}
	if len(c.Ports) != 4 {
		t.Errorf("expected tcp+udp per port, got %d", len(c.Ports))
	}
	svc := Service(gs, map[string]string{"lb.cilium.io/pool": "general"})
	if svc.Labels["lb.cilium.io/pool"] != "general" || len(svc.Spec.Ports) != 4 ||
		svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		t.Errorf("service = %+v", svc)
	}
	if svc.Spec.ExternalTrafficPolicy != corev1.ServiceExternalTrafficPolicyLocal {
		t.Errorf("default traffic policy: %s", svc.Spec.ExternalTrafficPolicy)
	}
	gs.Spec.ExternalTrafficPolicy = v1alpha1.TrafficCluster
	if p := Service(gs, nil).Spec.ExternalTrafficPolicy; p != corev1.ServiceExternalTrafficPolicyCluster {
		t.Errorf("traffic policy Cluster: %s", p)
	}
	// Explicit SingleStack (not nil), so switching IPv6 off releases the second family.
	if p := svc.Spec.IPFamilyPolicy; p == nil || *p != corev1.IPFamilyPolicySingleStack {
		t.Errorf("default IP family policy: %v", p)
	}
	gs.Spec.IPv6 = true
	if p := Service(gs, nil).Spec.IPFamilyPolicy; p == nil || *p != corev1.IPFamilyPolicyPreferDualStack {
		t.Errorf("IPv6 IP family policy: %v", p)
	}
}

func TestInstallPod(t *testing.T) {
	gs, e := testServer()
	e.Spec.Install = v1alpha1.InstallScript{
		Container:  "ghcr.io/pelican-eggs/installers:alpine",
		Entrypoint: "ash",
		Script:     "echo hi",
	}
	pod := InstallPod(gs, e, Options{})
	c := pod.Spec.Containers[0]
	// The image entrypoint stays; the shell runs the script, then hands the files to 988.
	if len(c.Command) != 0 || len(c.Args) != 4 || c.Args[0] != "ash" || c.Args[1] != "-c" || c.Args[3] != "ash" ||
		!strings.HasPrefix(c.Args[2], `"$0" /mnt/install/install.sh; rc=$?; chown -R -h 988:988 /mnt/server`) {
		t.Errorf("args = %q, command = %v", c.Args, c.Command)
	}
	if caps := c.SecurityContext.Capabilities; caps == nil || len(caps.Drop) != 1 || caps.Drop[0] != "ALL" {
		t.Errorf("install pod must drop all capabilities first: %+v", caps)
	}
	e.Spec.Install.Entrypoint = "python3"
	if args := InstallPod(gs, e, Options{}).Spec.Containers[0].Args; len(args) != 2 ||
		args[1] != "/mnt/install/install.sh" {
		t.Errorf("other entrypoints run the script directly: %q", args)
	}
	if mem := c.Resources.Limits[corev1.ResourceMemory]; mem.Value() != 2048<<20 {
		t.Errorf("install memory = %s", mem.String())
	}
	if pod.Spec.Affinity == nil || pod.Spec.Affinity.PodAffinity == nil {
		t.Error("install pod must run on the node of the files pod")
	}
}

func TestMemoryOverhead(t *testing.T) {
	for in, want := range map[int64]int64{1024: 1177, 2048: 2355, 4096: 4505, 8192: 8601} {
		if got := memoryLimitMiB(in); got != want {
			t.Errorf("memoryLimitMiB(%d) = %d, want %d", in, got, want)
		}
	}
}

func TestVolumeAffinity(t *testing.T) {
	gs, e := testServer()
	for role, pod := range map[string]*corev1.Pod{
		"game":    GamePod(gs, e, Options{}),
		"install": InstallPod(gs, e, Options{}),
		"files":   FilesPod(gs, Options{}),
	} {
		if pod.Labels[LabelVolume] != gs.Name {
			t.Errorf("%s pod lacks the volume label: %v", role, pod.Labels)
		}
		terms := pod.Spec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution
		// Each pod must match its own term, so the first one of the group can be scheduled.
		if len(terms) != 1 || terms[0].LabelSelector.MatchLabels[LabelVolume] != gs.Name ||
			terms[0].TopologyKey != "kubernetes.io/hostname" {
			t.Errorf("%s pod affinity = %+v", role, terms)
		}
	}
}

func TestResourceRequests(t *testing.T) {
	gs, e := testServer()
	gs.Spec.Resources.CPUMillis = 2000
	c := GamePod(gs, e, Options{}).Spec.Containers[0].Resources
	if c.Requests.Memory().Cmp(*c.Limits.Memory()) != 0 {
		t.Errorf("memory request %s must equal the limit %s", c.Requests.Memory(), c.Limits.Memory())
	}
	if c.Requests.Cpu().MilliValue() != 1000 || c.Limits.Cpu().MilliValue() != 2000 {
		t.Errorf("cpu request/limit = %s/%s, want half of the limit", c.Requests.Cpu(), c.Limits.Cpu())
	}
	gs.Spec.Resources.CPUMillis = 0
	c = GamePod(gs, e, Options{}).Spec.Containers[0].Resources
	if _, limited := c.Limits[corev1.ResourceCPU]; limited ||
		c.Requests.Cpu().MilliValue() != UnlimitedCPURequestMillis {
		t.Errorf("unlimited cpu: limits=%v requests=%v", c.Limits, c.Requests)
	}
	for name, pod := range map[string]*corev1.Pod{
		"install": InstallPod(gs, e, Options{}),
		"files":   FilesPod(gs, Options{}),
	} {
		r := pod.Spec.Containers[0].Resources
		if r.Limits.Cpu().IsZero() || r.Requests.Memory().Cmp(*r.Limits.Memory()) != 0 ||
			r.Requests.Cpu().MilliValue()*2 != r.Limits.Cpu().MilliValue() {
			t.Errorf("%s pod resources = %+v", name, r)
		}
	}
}

func TestRuntimeHash(t *testing.T) {
	gs, e := testServer()
	h := RuntimeHash(gs, e, Options{})
	if GamePod(gs, e, Options{}).Annotations[AnnotationRuntimeHash] != h {
		t.Error("game pod must carry the runtime hash")
	}
	same := gs.DeepCopy()
	same.Spec.DisplayName = "renamed"
	same.Spec.CrashRestart = nil
	if RuntimeHash(same, e, Options{}) != h {
		t.Error("display name and crash restart apply without restart")
	}
	for name, change := range map[string]func(*v1alpha1.GameServer){
		"memory":              func(g *v1alpha1.GameServer) { g.Spec.Resources.MemoryMiB++ },
		"cpu":                 func(g *v1alpha1.GameServer) { g.Spec.Resources.CPUMillis += 100 },
		"image":               func(g *v1alpha1.GameServer) { g.Spec.Image = "other" },
		"startup":             func(g *v1alpha1.GameServer) { g.Spec.Startup = "java -jar x" },
		"egg startup command": func(g *v1alpha1.GameServer) { g.Spec.StartupName = "Flags" },
		"variable":            func(g *v1alpha1.GameServer) { g.Spec.Environment = map[string]string{"X": "1"} },
		"ports":               func(g *v1alpha1.GameServer) { g.Spec.Ports = []int32{1} },
	} {
		c := gs.DeepCopy()
		change(c)
		if RuntimeHash(c, e, Options{}) == h {
			t.Errorf("%s change must require a restart", name)
		}
	}
	// Edits of the egg that apply at the next start.
	for name, change := range map[string]func(*v1alpha1.Egg){
		"egg startup": func(x *v1alpha1.Egg) { x.Spec.Startup = "./other" },
		"variable default": func(x *v1alpha1.Egg) {
			x.Spec.Variables = append(x.Spec.Variables, v1alpha1.EggVariable{EnvVariable: "NEW", DefaultValue: "1"})
		},
		"config files": func(x *v1alpha1.Egg) {
			x.Spec.ConfigFiles = []v1alpha1.ConfigFile{{File: "a.properties", Parser: "properties"}}
		},
	} {
		c := e.DeepCopy()
		change(c)
		if RuntimeHash(gs, c, Options{}) == h {
			t.Errorf("%s change must require a restart", name)
		}
	}
	// Changes that apply without a restart.
	live := e.DeepCopy()
	live.Spec.Description, live.Spec.Stop, live.Spec.Install.Script = "new", "^C", "echo new"
	if RuntimeHash(gs, live, Options{}) != h {
		t.Error("description, stop command and install script apply without restart")
	}
}

func TestFilesPodRunsAsGameUser(t *testing.T) {
	gs, _ := testServer()
	pod := FilesPod(gs, Options{HelperImage: "alpine"})
	sc := pod.Spec.Containers[0].SecurityContext
	if sc.RunAsUser == nil || *sc.RunAsUser != ServerUID || !*sc.RunAsNonRoot || !*sc.ReadOnlyRootFilesystem ||
		len(sc.Capabilities.Add) != 0 || sc.Capabilities.Drop[0] != "ALL" {
		t.Errorf("file operations must run as the game server user without privileges: %+v", sc)
	}
	init := pod.Spec.InitContainers[0]
	if strings.Join(init.Command, " ") != "chown -R -h 988:988 /home/container" {
		t.Errorf("ownership fix must not follow symbolic links: %q", init.Command)
	}
	if add := init.SecurityContext.Capabilities.Add; len(add) != 2 {
		t.Errorf("ownership fix capabilities: %v", add)
	}
}

func TestFixedIPs(t *testing.T) {
	v4, v6 := corev1.IPv4Protocol, corev1.IPv6Protocol
	for _, tc := range []struct {
		fixed    string
		families []corev1.IPFamily
		want     string
	}{
		{"192.0.2.170, 2001:db8::5", nil, "192.0.2.170,2001:db8::5"}, // a new service: all
		{"192.0.2.170, 2001:db8::5", []corev1.IPFamily{v4}, "192.0.2.170"},
		{"192.0.2.170, 2001:db8::5", []corev1.IPFamily{v4, v6}, "192.0.2.170,2001:db8::5"},
		{"2001:db8::5", []corev1.IPFamily{v4}, ""},
		{"", []corev1.IPFamily{v4}, ""},
	} {
		if got := FixedIPs(tc.fixed, tc.families); got != tc.want {
			t.Errorf("FixedIPs(%q, %v) = %q, want %q", tc.fixed, tc.families, got, tc.want)
		}
	}
}

func TestLoadBalancerIPsFollowFamilies(t *testing.T) {
	svc := &corev1.Service{}
	svc.Status.LoadBalancer.Ingress = []corev1.LoadBalancerIngress{
		{IP: "2001:db8::5"}, {Hostname: "lb"}, {IP: "192.0.2.5"},
	}
	svc.Spec.IPFamilies = []corev1.IPFamily{corev1.IPv4Protocol, corev1.IPv6Protocol}
	if got := LoadBalancerIPs(svc); !slices.Equal(got, []string{"192.0.2.5", "2001:db8::5"}) {
		t.Errorf("IPv4 first: %v", got)
	}
	svc.Spec.IPFamilies = []corev1.IPFamily{corev1.IPv6Protocol, corev1.IPv4Protocol}
	if got := LoadBalancerIPs(svc); !slices.Equal(got, []string{"2001:db8::5", "192.0.2.5"}) {
		t.Errorf("IPv6 first: %v", got)
	}
}
