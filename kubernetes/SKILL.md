# Skill: Kubernetes

## Overview

Kubernetes (K8s) is an open-source container orchestration system for automating deployment, scaling, and management of containerised applications. This skill covers core resource types, `kubectl` usage, and common operational tasks.

## Key Concepts

- **Cluster**: A set of nodes managed by a Kubernetes control plane.
- **Node**: A worker machine (VM or bare-metal) that runs Pods.
- **Pod**: The smallest deployable unit; wraps one or more containers.
- **Deployment**: Manages a replicated set of Pods with rolling-update support.
- **Service**: Exposes Pods on a stable DNS name and IP address.
- **Namespace**: A logical partition for isolating resources within a cluster.
- **ConfigMap / Secret**: Objects for injecting configuration and sensitive data into Pods.
- **Ingress**: Routes external HTTP/S traffic to Services.
- **PersistentVolumeClaim (PVC)**: Requests durable storage for a Pod.

## Common Tasks

### View cluster resources
```bash
kubectl get nodes
kubectl get pods -n <namespace>
kubectl get all -n <namespace>
kubectl describe pod <pod-name> -n <namespace>
```

### Apply a manifest
```bash
kubectl apply -f deployment.yaml
kubectl apply -f ./manifests/           # apply a directory
```

### Watch rollout status
```bash
kubectl rollout status deployment/<name> -n <namespace>
kubectl rollout history deployment/<name>
kubectl rollout undo deployment/<name>  # roll back
```

### Scale a Deployment
```bash
kubectl scale deployment/<name> --replicas=3 -n <namespace>
```

### Exec into a running Pod
```bash
kubectl exec -it <pod-name> -n <namespace> -- bash
```

### View logs
```bash
kubectl logs <pod-name> -n <namespace>
kubectl logs -f deployment/<name> -n <namespace>   # follow
kubectl logs <pod-name> --previous                 # crashed container
```

### Port-forward for local access
```bash
kubectl port-forward svc/<service> 8080:80 -n <namespace>
```

### Manage Secrets
```bash
kubectl create secret generic my-secret \
  --from-literal=password=s3cr3t \
  -n <namespace>
kubectl get secret my-secret -o jsonpath='{.data.password}' | base64 -d
```

### Delete resources
```bash
kubectl delete pod <pod-name> -n <namespace>
kubectl delete -f deployment.yaml
```

## Example Deployment Manifest

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  namespace: default
spec:
  replicas: 2
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
    spec:
      containers:
        - name: my-app
          image: ghcr.io/<owner>/my-app:1.0.0
          ports:
            - containerPort: 8080
          resources:
            requests:
              cpu: "100m"
              memory: "128Mi"
            limits:
              cpu: "500m"
              memory: "256Mi"
          readinessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
```

## Best Practices

- Always specify resource `requests` and `limits` for containers.
- Use namespaces to isolate workloads (e.g., `dev`, `staging`, `prod`).
- Store manifests in version control and apply them via CI/CD pipelines.
- Use `readinessProbe` and `livenessProbe` to enable automatic traffic management.
- Avoid running containers as root; set `securityContext.runAsNonRoot: true`.
- Use Secrets (not ConfigMaps) for sensitive values, and consider an external secrets operator.
- Label all resources consistently to enable filtering with `kubectl` and monitoring tools.

## References

- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [kubectl Cheat Sheet](https://kubernetes.io/docs/reference/kubectl/cheatsheet/)
- [Kubernetes Patterns Book](https://k8spatterns.io/)
