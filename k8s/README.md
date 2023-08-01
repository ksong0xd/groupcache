# groupcache/k8s

groupcache/k8s introduces the mechanism to dynamically manage the list of peers
of the group cache based on the registered services.

## setup

The groupcache/k8s relies on a configuration that the endpoints of the groupcache
participants are under a single kubernetes
[Service](https://kubernetes.io/docs/concepts/services-networking/service/).

Also, k8s package requires to have the following parameters:
- the pod IP address of itself; it can be specified through an environment
  variable with [fieldRef](https://kubernetes.io/docs/tasks/inject-data-application/environment-variable-expose-pod-information/)
- the port number for the groupcache.

The initialization code will be:

```go
m, err := k8s.NewPeersManager(
    ctx,
    client,
    serviceName,
    namespace,
    port,
    fmt.Sprintf("%s:%d", self, port),
)
```

Note that the client is a kubernetes client, which would be typically obtained
through `InClusterConfig`.

By default, the peers manager will set up non-TLS connection among peers, but
you could specify connection options by adding extra groupcache.GRPCPoolOption
parameters.

`NewPeersManager` sets up a gRPC peer-picker and also watches endpoint slices
to update the peers when changes happen. After the peers manager is created,
all you will have to do is to set up a `groupcache.Group` and combine it with
your application code.

See [./example/cmd/main.go](./example/cmd/main.go) for an example.

## permissions

This package observes [EndpointSlice](https://kubernetes.io/docs/concepts/services-networking/endpoint-slices/)
to produce the list of peers, but it is not allowed by default. You will have to
set up a service account, role, and rolebinding for that.

The service account doesn't need any specific content.
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: <name>
```

The role will specify the consumption of endpoint slices, and role-binding will
bind the role and the service account.

```yaml
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: <name>
rules:
- apiGroups: ["discovery.k8s.io"]
  resources: ["endpointslices"]
  verbs: ["get", "watch", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: <name>
subjects:
- kind: ServiceAccount
  name: <service-account name>
roleRef:
  kind: Role
  name: <role name>
  apiGroup: rbac.authorization.k8s.io
```

Finally your workload (e.g. podTemplate spec in deployment) needs to specify
the service account.

```yaml
spec:
  containers:
    ...
  serviceAccount: <service-account name>
```

See also [the example YAML file](./example/fib.yaml).