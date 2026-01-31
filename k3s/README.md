# Kubernetes (k3s) Deployment

This directory contains the manifests to deploy the backend infrastructure to a Kubernetes cluster (e.g., k3s).

## structure

- `api/`: Backend API deployment and service
- `postgres/`: Database StatefulSet, Service, and PVC
- `ingress/`: Ingress configuration (traefik/nginx)
- `config/`: Configuration and Secrets

## Prerequisites

- A running Kubernetes cluster (k3s recommended)
- `kubectl` configured to talk to your cluster
- Container image for the API built and pushed to a registry (or available locally in k3s)

## Deployment Steps

1. **Secrets Configuration**
   Copy the example secrets and edit them with your actual values:
   ```bash
   cp k3s/config/secrets.example.yaml k3s/config/secrets.yaml
   # Edit secrets.yaml with real passwords and keys
   # Base64 encoding is handled if you use stringData, otherwise encode values manually.
   ```
   Apply the config and secrets:
   ```bash
   kubectl apply -f k3s/config/
   ```

2. **Database**
   Deploy Postgres:
   ```bash
   kubectl apply -f k3s/postgres/
   ```

3. **Backend API**
   Ensure your image is built. If using k3s and building locally:
   ```bash
   docker build -t ghcr.io/rakaoran/gto-backend:latest -f backend/api/Dockerfile.production backend/api
   docker save ghcr.io/rakaoran/gto-backend:latest | sudo k3s ctr images import -
   ```
   Then deploy:
   ```bash
   kubectl apply -f k3s/api/
   ```

4. **Ingress**
   Apply the ingress rules:
   ```bash
   kubectl apply -f k8s/ingress/
   ```

## Notes

- **SSL/TLS**: The ingress assumes you have a certificate manager or are handling SSL at the load balancer level. If using `cert-manager`, ensure the `ClusterIssuer` matches the annotation.
- **Persistence**: Postgres data is persisted to a PVC. Ensure your cluster has a default StorageClass or update `pvc.yaml` to specify one.
