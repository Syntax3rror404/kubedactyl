package tenancy

import "testing"

func TestOwns(t *testing.T) {
	SystemNamespace, NamespacePrefix = "kubedactyl", "kubedactyl-user-"
	for ns, want := range map[string]bool{
		"kubedactyl": true, "kubedactyl-user-alice": true,
		"kubedactyl-uninstalltest": false, "kubedactyl-uninstalltest-user-alice": false, "default": false,
	} {
		if Owns(ns) != want {
			t.Errorf("Owns(%q) = %v, want %v", ns, !want, want)
		}
	}
	if Namespace("bob") != "kubedactyl-user-bob" {
		t.Error(Namespace("bob"))
	}
}
