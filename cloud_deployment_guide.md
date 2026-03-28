# Cloud Deployment Guide: Taking Your App Live 🌐

Now that we have fully secured our application with **bcrypt hashing** and robust **authentication layers**, it is time to deploy it to the actual internet.

Here is exactly how you move from a local `kind` cluster to a public Kubernetes provider (like **DigitalOcean**, **Google Kubernetes Engine (GKE)**, or **Linode**).

## 1. Prepare and Push Docker Images

Currently, `build_and_deploy.ps1` builds images locally and uses `kind load docker-image` to push them to your local cluster.

For the live internet, we need a public or private container registry like Docker Hub or GitHub Container Registry (GHCR).

**A. Login to your container registry:**
```bash
docker login docker.io
# OR
docker login ghcr.io
```

**B. Tag your images for your registry:**
Substitute `YOUR_USERNAME` with your Docker Hub username.
```bash
docker build -t YOUR_USERNAME/api-gateway:latest -f ./services/api-gateway/Dockerfile .
docker build -t YOUR_USERNAME/auth-service:latest -f ./services/auth/Dockerfile .
docker build -t YOUR_USERNAME/order-service:latest -f ./services/order/Dockerfile .
docker build -t YOUR_USERNAME/inventory-service:latest -f ./services/inventory/Dockerfile .
docker build -t YOUR_USERNAME/notification-service:latest -f ./services/notification/Dockerfile .
```

**C. Push the images to the public registry:**
```bash
docker push YOUR_USERNAME/api-gateway:latest
docker push YOUR_USERNAME/auth-service:latest
docker push YOUR_USERNAME/order-service:latest
docker push YOUR_USERNAME/inventory-service:latest
docker push YOUR_USERNAME/notification-service:latest
```

## 2. Update Kubernetes Manifests

In the cloud, your nodes aren't pulling from Local. They must pull from your exact internet URLs.

**A. Modify `k8s/applications/` descriptors**

Go into each `deployment.yaml` inside your `k8s/applications/` and change the `image` string:
```diff
- image: gcr.io/local-dev/api-gateway:latest
+ image: YOUR_USERNAME/api-gateway:latest
```

**B. Change `imagePullPolicy`**
Whenever you push new tags without changing the `latest` version name, Kubernetes won't know to pull the new version unless you set:
```yaml
imagePullPolicy: Always
```
Ensure this is set for your container specs so the cloud cluster actively fetches your new images.

## 3. Provisioning Cloud Infrastructure

You will need a Managed Kubernetes instance (which is usually much cheaper and easier than maintaining your own masters/nodes).
1. Create a cluster on **DigitalOcean Kubernetes** or **GKE**.
2. Download the `kubeconfig.yaml` file provided by the cloud interface.
3. Merge it into your local `~/.kube/config`, or set the context via:
```bash
export KUBECONFIG=~/Downloads/k8s-cluster-kubeconfig.yaml
```
4. Verify you're connected to the live internet cluster:
```bash
kubectl get nodes
```

## 4. Spin up the World! 🌍

Once your context is aimed at the cloud cluster and your manifests are pointing to your public Docker Hub images, you simply apply the same commands as `build_and_deploy.ps1`:

```bash
kubectl apply -f k8s/config/
kubectl apply -f k8s/databases/
kubectl apply -f k8s/applications/
kubectl apply -f k8s/networking/
```

**Ingress & IP Addresses:**
When deploying to AWS, GCP, or DigitalOcean, the `ingress-nginx` controller will automatically request a real, publicly routable Load Balancer IP. Find it by running:
```bash
kubectl get services -n ingress-nginx
```
Wait for the `EXTERNAL-IP` to populate, and navigate to that IP in your browser to see your live, secured microservices dashboard!
