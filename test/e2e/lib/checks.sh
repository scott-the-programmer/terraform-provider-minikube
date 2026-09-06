# shellcheck shell=bash
#
# The individual e2e assertions. Each function returns 0 on success and prints
# diagnostics to stderr on failure; run.sh wraps them with `check` so the whole
# list runs even when one of them fails.

E2E_NS=e2e
E2E_SVC=nginx
E2E_SENTINEL='terraform-provider-minikube e2e ok'

# Trailing dot on purpose: it makes the name absolute, so the resolver never
# walks the pod's search list. Those entries are inherited from the host, and a
# host search domain that SERVFAILs makes musl (alpine) give up on the whole
# lookup rather than move on to the next candidate.
E2E_SVC_FQDN="$E2E_SVC.$E2E_NS.svc.cluster.local."

# --- terraform surface -------------------------------------------------------

checks::outputs_usable() {
  local tf_dir=$1 host
  host=$(tf::output "$tf_dir" host) || return 1
  assert::not_empty "$host" || return 1
  assert::contains "$host" 'https://' || return 1

  local field
  for field in cluster_ca_certificate client_certificate client_key; do
    local value
    value=$(tf::output "$tf_dir" "$field") || return 1
    assert::contains "$value" '-----BEGIN' || {
      printf 'output %s does not look like PEM\n' "$field" >&2
      return 1
    }
  done
}

# --- cluster health ----------------------------------------------------------

checks::apiserver_reachable() {
  local out
  out=$(retry::until 120 5 kube cluster-info) || return 1
  assert::contains "$out" 'Kubernetes control plane'
}

checks::nodes_ready() {
  local expected=$1
  retry::until 300 5 _checks::nodes_ready_once "$expected" >/dev/null
}

_checks::nodes_ready_once() {
  local expected=$1 ready total
  total=$(kube get nodes --no-headers 2>/dev/null | grep -c . || true)
  ready=$(kube get nodes \
    -o jsonpath='{range .items[*]}{range .status.conditions[?(@.type=="Ready")]}{.status}{"\n"}{end}{end}' 2>/dev/null |
    grep -c '^True$' || true)
  if [[ "$total" != "$expected" || "$ready" != "$expected" ]]; then
    printf 'expected %s ready nodes, have %s ready of %s\n' "$expected" "$ready" "$total" >&2
    return 1
  fi
}

checks::system_pods_ready() {
  # coredns is the one that actually matters for the service DNS assertion
  # below, so wait on the whole control plane namespace rather than guessing.
  kube -n kube-system wait --for=condition=Ready pods --all --timeout=300s >/dev/null
}

checks::default_storageclass() {
  local out
  out=$(kube get storageclass -o jsonpath='{.items[*].metadata.name}') || return 1
  assert::contains "$out" 'standard'
}

# --- workload ----------------------------------------------------------------

checks::apply_manifests() {
  local manifest=$1
  kube apply -f "$manifest" >/dev/null
}

checks::rollout_complete() {
  kube -n "$E2E_NS" rollout status deployment/nginx --timeout=300s >/dev/null
}

checks::pods_running() {
  local expected=$1 running
  running=$(kube -n "$E2E_NS" get pods -l app=nginx \
    --field-selector=status.phase=Running --no-headers | grep -c . || true)
  assert::equals "$expected" "$running"
}

checks::endpoints_ready() {
  local expected=$1
  retry::until 120 3 _checks::endpoints_once "$expected" >/dev/null
}

_checks::endpoints_once() {
  local expected=$1 actual
  actual=$(kube::ready_endpoints "$E2E_NS" "$E2E_SVC")
  if [[ "$actual" != "$expected" ]]; then
    printf 'expected %s ready endpoints for svc/%s, have %s\n' \
      "$expected" "$E2E_SVC" "$actual" >&2
    return 1
  fi
}

# Cluster DNS has to resolve the service to its ClusterIP, from inside a pod.
checks::cluster_dns() {
  local cluster_ip out
  cluster_ip=$(kube -n "$E2E_NS" get svc "$E2E_SVC" -o jsonpath='{.spec.clusterIP}') || return 1
  assert::not_empty "$cluster_ip" || return 1

  out=$(retry::until 120 5 kube -n "$E2E_NS" exec deploy/nginx -- \
    nslookup "$E2E_SVC_FQDN") || return 1
  assert::contains "$out" "$cluster_ip"
}

# Pod -> Service -> Pod, entirely inside the cluster. busybox wget ships in the
# nginx alpine image, so this needs no extra pull.
checks::in_cluster_http() {
  local out
  out=$(retry::until 120 5 kube -n "$E2E_NS" exec deploy/nginx -- \
    wget -q -O - -T 10 "http://$E2E_SVC_FQDN/") || return 1
  assert::contains "$out" "$E2E_SENTINEL"
}

# Host -> apiserver -> pod. Works for every driver because the traffic is
# tunnelled through the same apiserver connection kubectl already proved.
checks::port_forward_http() {
  local log=$1 body rc=0
  kube::port_forward "$E2E_NS" "$E2E_SVC" 80 "$log" || return 1

  body=$(retry::until 60 3 curl -sS --max-time 10 \
    "http://127.0.0.1:$E2E_PORT_FORWARD_PORT/") || rc=1
  if [[ $rc -eq 0 ]]; then
    assert::contains "$body" "$E2E_SENTINEL" || rc=1
  fi

  kube::stop_port_forward
  return $rc
}

# Host -> node IP -> NodePort. Only routable for the container drivers on Linux,
# so the caller probes reachability first.
checks::nodeport_reachable() {
  local node_ip=$1
  [[ -n $node_ip ]] || return 1
  curl -sS -o /dev/null --connect-timeout 5 --max-time 10 "http://$node_ip:30080/" 2>/dev/null
}

checks::nodeport_http() {
  local node_ip=$1 body
  body=$(retry::until 60 3 curl -sS --max-time 10 "http://$node_ip:30080/") || return 1
  assert::contains "$body" "$E2E_SENTINEL"
}

# --- lifecycle ---------------------------------------------------------------

checks::scale_to() {
  local replicas=$1
  kube -n "$E2E_NS" scale deployment/nginx --replicas="$replicas" >/dev/null || return 1
  kube -n "$E2E_NS" rollout status deployment/nginx --timeout=300s >/dev/null || return 1
  _checks::endpoints_once "$replicas" >/dev/null ||
    retry::until 120 3 _checks::endpoints_once "$replicas" >/dev/null
}

checks::workload_removed() {
  kube delete namespace "$E2E_NS" --timeout=180s >/dev/null || return 1
  local out
  out=$(kube get namespace "$E2E_NS" 2>&1) && {
    printf 'namespace %s still present:\n%s\n' "$E2E_NS" "$out" >&2
    return 1
  }
  return 0
}

checks::cluster_gone() {
  local tf_dir=$1 remaining
  remaining=$(tf::state_count "$tf_dir")
  if [[ "$remaining" != "0" ]]; then
    printf 'expected an empty state after destroy, %s resources remain\n' "$remaining" >&2
    return 1
  fi

  # The apiserver the provider handed us must no longer answer.
  if kubectl --kubeconfig "$E2E_KUBECONFIG" --request-timeout=10s get nodes >/dev/null 2>&1; then
    printf 'apiserver still reachable after terraform destroy\n' >&2
    return 1
  fi
  return 0
}
