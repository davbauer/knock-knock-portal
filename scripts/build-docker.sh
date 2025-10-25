#!/bin/bash
# Multi-platform Docker build script for Knock-Knock Portal

set -e

echo "========================================="
echo "Knock-Knock Portal - Multi-Platform Build"
echo "========================================="
echo ""

# Check if buildx is available
if ! docker buildx version &> /dev/null; then
    echo "Error: docker buildx is not available!"
    echo "Please enable Docker BuildKit"
    exit 1
fi

# Create/use buildx builder
echo "Setting up buildx builder..."
if ! docker buildx inspect multiplatform &> /dev/null; then
    echo "Creating new buildx builder..."
    docker buildx create --name multiplatform --driver docker-container --bootstrap --use
else
    echo "Using existing buildx builder..."
    docker buildx use multiplatform
fi

echo ""
echo "Available platforms:"
docker buildx inspect --bootstrap | grep "Platforms:"

echo ""
echo "========================================="
echo "Choose build option:"
echo "========================================="
echo "1) Single platform (current architecture)"
echo "2) Multi-platform (amd64 + arm64)"
echo "3) AMD64 only"
echo "4) ARM64 only"
echo ""
read -p "Enter choice [1-4]: " choice

case $choice in
    1)
        echo ""
        echo "Building for current platform..."
        docker build -t knock-knock-portal:latest .
        ;;
    2)
        echo ""
        echo "Building for linux/amd64 and linux/arm64..."
        read -p "Push to registry? (y/n): " push
        if [[ "$push" == "y" ]]; then
            read -p "Enter registry/image name (e.g., user/knock-knock-portal): " IMAGE_NAME
            docker buildx build \
                --platform linux/amd64,linux/arm64 \
                --tag ${IMAGE_NAME}:latest \
                --push \
                .
        else
            docker buildx build \
                --platform linux/amd64,linux/arm64 \
                --tag knock-knock-portal:latest \
                --load \
                .
        fi
        ;;
    3)
        echo ""
        echo "Building for linux/amd64 only..."
        docker buildx build \
            --platform linux/amd64 \
            --tag knock-knock-portal:latest \
            --load \
            .
        ;;
    4)
        echo ""
        echo "Building for linux/arm64 only..."
        docker buildx build \
            --platform linux/arm64 \
            --tag knock-knock-portal:latest \
            --load \
            .
        ;;
    *)
        echo "Invalid choice!"
        exit 1
        ;;
esac

echo ""
echo "========================================="
echo "Build complete!"
echo "========================================="
echo ""
echo "Image created: knock-knock-portal:latest"
echo ""
echo "To run the container:"
echo "  docker-compose up -d"
echo ""
echo "Or manually:"
echo "  docker run -p 8000:8000 -p 8080-8099:8080-8099 \\"
echo "    -v \$(pwd)/backend/config.yml:/app/config/config.yml:ro \\"
echo "    -e ADMIN_PASSWORD_BCRYPT_HASH=\${ADMIN_PASSWORD_BCRYPT_HASH} \\"
echo "    -e JWT_SIGNING_SECRET_KEY=\${JWT_SIGNING_SECRET_KEY} \\"
echo "    knock-knock-portal:latest"
echo ""
