$ErrorActionPreference = "Stop"

$services = @(
    @{ dir="api-gateway"; tag="api-gateway" },
    @{ dir="auth"; tag="auth-service" },
    @{ dir="inventory"; tag="inventory-service" },
    @{ dir="order"; tag="order-service" },
    @{ dir="notification"; tag="notification-service" }
)

Write-Host "========================================="
Write-Host "Building Docker Images for Microservices"
Write-Host "========================================="
foreach ($svc in $services) {
    $dir = $svc.dir
    $tag = $svc.tag
    Write-Host "Building $dir as $tag..."
    docker build -t "gcr.io/local-dev/${tag}:latest" -f "./services/$dir/Dockerfile" .
    
    Write-Host "Loading $tag into kind cluster..."
    kind load docker-image "gcr.io/local-dev/${tag}:latest"
}

Write-Host "`n========================================="
Write-Host "Applying Kubernetes Manifests"
Write-Host "========================================="
kubectl apply -f k8s/config/
kubectl apply -f k8s/databases/
kubectl apply -f k8s/applications/
kubectl apply -f k8s/networking/

Write-Host "`n========================================="
Write-Host "Deployment complete! Next steps:"
Write-Host "========================================="
Write-Host "1. Wait for pods to be ready by running: kubectl get pods -w"
Write-Host "2. Send requests to the Ingress (usually localhost:80 on Docker Desktop) or port-forward the API gateway:"
Write-Host "   kubectl port-forward svc/api-gateway 8080:80"
