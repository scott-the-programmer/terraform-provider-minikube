# shellcheck shell=bash
#
# kubectl helpers. The kubeconfig is built purely from terraform outputs, which
# is the point of the exercise: if the provider does not surface a working
# endpoint and client certificate, nothing below can connect.

# kube::write_kubeconfig <tf_dir> <kubeconfig_path> <context_name>
kube::write_kubeconfig() {
  local tf_dir=$1 path=$2 name=$3
  local host ca cert key

  host=$(tf::output "$tf_dir" host)
  ca=$(tf::output "$tf_dir" cluster_ca_certificate | base64 | tr -d '\n')
  cert=$(tf::output "$tf_dir" client_certificate | base64 | tr -d '\n')
  key=$(tf::output "$tf_dir" client_key | base64 | tr -d '\n')

  local previous_umask
  previous_umask=$(umask)
  umask 077
  cat >"$path" <<EOF
apiVersion: v1
kind: Config
clusters:
  - name: $name
    cluster:
      server: $host
      certificate-authority-data: $ca
users:
  - name: $name
    user:
      client-certificate-data: $cert
      client-key-data: $key
contexts:
  - name: $name
    context:
      cluster: $name
      user: $name
current-context: $name
EOF
  umask "$previous_umask"
}

# kube <kubectl args...>
kube() {
  kubectl --kubeconfig "$E2E_KUBECONFIG" --request-timeout=30s "$@"
}

# kube::ready_endpoints <namespace> <service>
#
# Counts addresses across all EndpointSlices for the service. Empty slices count
# as zero rather than erroring.
kube::ready_endpoints() {
  local ns=$1 svc=$2
  kube -n "$ns" get endpointslices \
    -l "kubernetes.io/service-name=$svc" \
    -o jsonpath='{range .items[*]}{range .endpoints[?(@.conditions.ready==true)]}{.addresses[0]}{"\n"}{end}{end}' |
    grep -c . || true
}

# kube::node_ip - the address kubectl believes the first node is reachable on.
kube::node_ip() {
  kube get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}'
}

# kube::port_forward <namespace> <service> <remote_port> <log_file>
#
# Starts a background port-forward on an ephemeral local port. On success sets
# E2E_PORT_FORWARD_PORT and E2E_PORT_FORWARD_PID; the caller must then call
# kube::stop_port_forward. Every failure path cleans up after itself, so a
# forward is never left running.
#
# The port is returned through a global rather than stdout so the caller does
# not have to use a command substitution, which would put the background
# process in a subshell nobody can reap.
kube::port_forward() {
  local ns=$1 svc=$2 port=$3 log=$4

  : >"$log"
  kubectl --kubeconfig "$E2E_KUBECONFIG" \
    -n "$ns" port-forward "svc/$svc" ":$port" --address 127.0.0.1 >"$log" 2>&1 &
  E2E_PORT_FORWARD_PID=$!
  E2E_PORT_FORWARD_PORT=''

  local deadline=$((SECONDS + 30))
  while ((SECONDS < deadline)); do
    if ! kill -0 "$E2E_PORT_FORWARD_PID" 2>/dev/null; then
      printf 'port-forward exited early:\n%s\n' "$(cat "$log")" >&2
      E2E_PORT_FORWARD_PID=''
      return 1
    fi
    E2E_PORT_FORWARD_PORT=$(sed -n 's#^Forwarding from 127\.0\.0\.1:\([0-9]\+\).*#\1#p' "$log" | head -1)
    [[ -n $E2E_PORT_FORWARD_PORT ]] && return 0
    sleep 1
  done

  printf 'port-forward never reported a local port:\n%s\n' "$(cat "$log")" >&2
  kube::stop_port_forward
  return 1
}

kube::stop_port_forward() {
  if [[ -n ${E2E_PORT_FORWARD_PID:-} ]] && kill -0 "$E2E_PORT_FORWARD_PID" 2>/dev/null; then
    kill "$E2E_PORT_FORWARD_PID" 2>/dev/null || true
    wait "$E2E_PORT_FORWARD_PID" 2>/dev/null || true
  fi
  E2E_PORT_FORWARD_PID=''
}
