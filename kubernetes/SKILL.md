---
name: kubernetes
description: Kubernetes cluster management with k9s, kubectx, and kubectl-ai
---

# Kubernetes Management Skill

## Quick Commands

### Context Switching (kubectx)
```bash
kubectx                    # List contexts
kubectx <context>          # Switch context
kubectx -                  # Switch to previous
```

### Namespace Switching (kubens)
```bash
kubens                     # List namespaces
kubens <namespace>         # Switch namespace
kubens -                   # Switch to previous
```

### k9s Terminal UI
```bash
k9s                        # Launch k9s
k9s -n <namespace>         # Launch in namespace
k9s -c pods                # Jump to pods view
k9s -c svc                 # Jump to services
k9s -c deploy              # Jump to deployments
```

### Common kubectl patterns
```bash
kubectl get pods -o wide
kubectl logs -f <pod>
kubectl exec -it <pod> -- sh
kubectl describe pod <pod>
kubectl top pods
```

## Tundra Dome Specific

```bash
# Check tundra-dome namespace
kubens tundra-dome
k9s -n tundra-dome

# Airflow scheduler
kubectl exec -it deploy/tundra-dome-critical-airflow-scheduler -- airflow dags list

# Kafka topics
kubectl exec -it deploy/tundra-dome-critical-kafka -- kafka-topics.sh --list --bootstrap-server localhost:9092
```
