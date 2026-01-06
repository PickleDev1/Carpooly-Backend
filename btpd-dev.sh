#!/bin/bash

echo "Deploying to DEVELOPMENT environment..."

# Use existing builder if available, create only if needed
if ! docker buildx inspect amd64-builder >/dev/null 2>&1; then
    echo "Creating new builder..."
    docker buildx create --name amd64-builder --driver docker-container --bootstrap
fi
docker buildx use amd64-builder

# Set default platform
export DOCKER_DEFAULT_PLATFORM=linux/amd64

# Build and push to dev image
DOCKER_BUILDKIT=1 docker buildx build --platform=linux/amd64 --push -t gcr.io/carpooly-login/car-backend-dev .
sleep 30;

# Deploy to development environment
gcloud run deploy car-backend-dev \
    --image gcr.io/carpooly-login/car-backend-dev \
    --add-cloudsql-instances carpooly-login:us-west1:carpool-dev1 \
    --project carpooly-login \
    --region us-west1 \
    --service-account="car-backend-service-account@carpooly-login.iam.gserviceaccount.com" \
    --set-env-vars INSTANCE_CONNECTION_NAME=carpooly-login:us-west1:carpool-dev1 \
    --set-env-vars DB_USER=postgres \
    --set-env-vars DB_NAME=carpool_dev \
    --set-env-vars DB_HOST=34.168.6.42 \
    --set-env-vars DB_PORT=5432 \
    --set-env-vars CLERK_SECRET_KEY="sk_test_fJhTPSQYgBH6K1Y91xI6ZOUT2Z47D02ZU9N5oJ23Nw" \
    --set-env-vars RESEND_API_KEY="re_GuUg41fC_Cbfs6DXWCYv7L4LpnEWTw34G" \
    --set-env-vars WEBHOOK_SECRET="whsec_za7e7eVUHpdG9qhUn/AYkyWZ6A4urfEz" \
    --set-env-vars ENV=dev \
    --set-secrets DB_PASSWORD=db-password:latest \
    --min-instances 1 \
    --timeout 300s \
    --cpu 1 \
    --memory 512Mi

echo "Development deployment completed!"
echo "Development API URL: https://car-backend-dev-884945568547.us-west1.run.app"
echo "Set this as NEXT_PUBLIC_API_URL in your Vercel dev environment variables" 