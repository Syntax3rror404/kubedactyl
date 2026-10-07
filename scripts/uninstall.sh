#!/usr/bin/env bash
# Removes everything a Kubedactyl installation created in a cluster: the panel namespace
# (eggs, users, auth and OIDC secrets), every user namespace "<namespace>-user-<name>" with its game
# servers, pods, services and volumes (the data is deleted) and, with DELETE_CRDS=1, the CRDs.
# Works without a running panel.
#
#   KUBE_CONTEXT=my-cluster scripts/uninstall.sh            # asks for confirmation
#   YES=1 KUBEDACTYL_NAMESPACE=other DELETE_CRDS=1 scripts/uninstall.sh
set -euo pipefail

CTX=${KUBE_CONTEXT:?KUBE_CONTEXT must be set}
NS=${KUBEDACTYL_NAMESPACE:-kubedactyl}
DELETE_CRDS=${DELETE_CRDS:-0}
k() { kubectl --context "$CTX" "$@"; }

echo "Context:   $CTX"
# User namespaces of this installation: labeled by the panel and named <namespace>-user-<user>.
USER_NS=$(k get ns -l kubedactyl.io/user -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' | grep "^$NS-user-" || true)
ALL_NS="$NS $USER_NS"

echo "Namespace: $NS (eggs, users and everything below will be deleted)"
for n in $USER_NS; do echo "           $n (user namespace with game servers and data)"; done
[ "$DELETE_CRDS" = 1 ] && echo "CRDs:      eggs, gameservers, users, panelsettings, invites (kubedactyl.io), affects ALL installations"
if [ "${YES:-0}" != 1 ]; then
  read -r -p "Type the namespace name to confirm: " answer
  [ "$answer" = "$NS" ] || { echo "Aborted."; exit 1; }
fi

pvs() {
  for n in $ALL_NS; do
    k get pv -o jsonpath="{range .items[?(@.spec.claimRef.namespace==\"$n\")]}{.metadata.name}{\"\\n\"}{end}"
  done
}

# 1. The "longhorn" storage class retains volumes; switch them to Delete so the data goes away.
for pv in $(pvs); do
  k patch pv "$pv" --type merge -p '{"spec":{"persistentVolumeReclaimPolicy":"Delete"}}' >/dev/null
  echo "PV $pv: reclaim policy set to Delete"
done

# 2. Release the finalizers normally handled by the panel (it may not be running).
for n in $ALL_NS; do
  for kind in gameservers.kubedactyl.io users.kubedactyl.io; do
    k get crd "$kind" >/dev/null 2>&1 || continue
    for obj in $(k -n "$n" get "$kind" -o name 2>/dev/null); do
      k -n "$n" patch "$obj" --type merge -p '{"metadata":{"finalizers":null}}' >/dev/null
      echo "$n/$obj: finalizer released"
    done
  done
done

# 3. The namespaces take pods, services, PVCs, config maps, eggs, users and game servers with them.
for n in $ALL_NS; do
  k delete namespace "$n" --ignore-not-found --wait=false
done
for n in $ALL_NS; do
  k wait --for=delete "namespace/$n" --timeout=5m 2>/dev/null || true
done

# 4. Wait until the volumes are gone (Released PVs are deleted by the provisioner).
for _ in $(seq 1 60); do
  [ -z "$(pvs)" ] && break
  sleep 2
done
left=$(pvs)
[ -n "$left" ] && echo "WARNING: volumes still present: $left" || echo "All volumes deleted."

# 5. Optionally remove the CRDs (they are recreated by the panel on its next start).
if [ "$DELETE_CRDS" = 1 ]; then
  k delete crd eggs.kubedactyl.io gameservers.kubedactyl.io users.kubedactyl.io panelsettings.kubedactyl.io invites.kubedactyl.io --ignore-not-found
fi
echo "Done."
