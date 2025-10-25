# Knock-Knock Portal - Docker Deployment

## Quick Start

### 1. Build the Docker Image

```bash
docker build -t knock-knock-portal:latest .
```

**Build time:** ~3-5 minutes (first build)  
**Image size:** ~30MB (Alpine-based minimal image)

### 2. Prepare Configuration

Create your configuration file:

```bash
cp backend/config.example.yml my-config.yml
```

Edit `my-config.yml` with your settings.

### 3. Set Required Environment Variables

```bash
# Admin password (bcrypt hash)
export ADMIN_PASSWORD_BCRYPT_HASH='$2a$10$...'

# JWT signing secret (generate with: openssl rand -base64 32)
export JWT_SIGNING_SECRET_KEY='your-secret-key-here'
```

### 4. Run the Container

```bash
docker run -d \
  --name knock-knock-portal \
  -p 8000:8000 \
  -p 8080-8099:8080-8099 \
  -v $(pwd)/my-config.yml:/app/config/config.yml:ro \
  -e ADMIN_PASSWORD_BCRYPT_HASH="${ADMIN_PASSWORD_BCRYPT_HASH}" \
  -e JWT_SIGNING_SECRET_KEY="${JWT_SIGNING_SECRET_KEY}" \
  knock-knock-portal:latest
```

### 5. Verify Deployment

```bash
# Check health
curl http://localhost:8000/api/health

# Check logs
docker logs knock-knock-portal

# Follow logs
docker logs -f knock-knock-portal
```

---

## Docker Compose Deployment

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  knock-knock-portal:
    image: knock-knock-portal:latest
    build:
      context: .
      dockerfile: Dockerfile
    container_name: knock-knock-portal
    restart: unless-stopped
    
    ports:
      # Admin/Portal API
      - "8000:8000"
      # Proxy services (adjust based on your config)
      - "8080-8099:8080-8099"
    
    volumes:
      # Mount your config file (read-only)
      - ./my-config.yml:/app/config/config.yml:ro
      # Optional: Mount logs directory
      - ./logs:/app/logs
    
    environment:
      # Required secrets (use .env file or secrets)
      - ADMIN_PASSWORD_BCRYPT_HASH=${ADMIN_PASSWORD_BCRYPT_HASH}
      - JWT_SIGNING_SECRET_KEY=${JWT_SIGNING_SECRET_KEY}
      
      # Optional: Set timezone
      - TZ=America/New_York
    
    healthcheck:
      test: ["CMD", "/app/knock-knock", "--health-check"]
      interval: 30s
      timeout: 3s
      start_period: 5s
      retries: 3
    
    # Security: Run as non-root user
    user: "1000:1000"
    
    # Security: Drop capabilities
    cap_drop:
      - ALL
    
    # Security: Read-only root filesystem (except logs)
    read_only: true
    tmpfs:
      - /tmp:noexec,nosuid,size=50M
```

Run with:

```bash
docker-compose up -d
```

---

## Production Deployment with Secrets

### Using Docker Secrets (Swarm/Compose)

```yaml
version: '3.8'

services:
  knock-knock-portal:
    image: knock-knock-portal:latest
    
    secrets:
      - admin_password_hash
      - jwt_secret_key
    
    environment:
      - ADMIN_PASSWORD_BCRYPT_HASH_FILE=/run/secrets/admin_password_hash
      - JWT_SIGNING_SECRET_KEY_FILE=/run/secrets/jwt_secret_key

secrets:
  admin_password_hash:
    file: ./secrets/admin_password_hash.txt
  jwt_secret_key:
    file: ./secrets/jwt_secret_key.txt
```

### Using .env File

Create `.env`:

```bash
ADMIN_PASSWORD_BCRYPT_HASH=$2a$10$hv1bj.Z0tDFRGepS5BvARO.NBrbsXmA8qAD4jnrCdhqeldYRKLbZW
JWT_SIGNING_SECRET_KEY=your-secret-key-here
```

Add to `.gitignore`:
```
.env
my-config.yml
secrets/
```

---

## Build Arguments

### Multi-Architecture Build

```bash
# Build for multiple platforms
docker buildx build --platform linux/amd64,linux/arm64 -t knock-knock-portal:latest .
```

### Development Build (with debug symbols)

Modify Dockerfile build stage:

```dockerfile
# Development build (larger but with debug info)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -o knock-knock \
    ./cmd/server/main.go
```

---

## Volume Mounts

### Required Volumes

1. **Configuration** (required):
   ```bash
   -v $(pwd)/my-config.yml:/app/config/config.yml:ro
   ```

2. **Logs** (optional):
   ```bash
   -v $(pwd)/logs:/app/logs
   ```

### File Permissions

The container runs as user `knockknock` (UID 1000, GID 1000).

Ensure your mounted volumes have correct permissions:

```bash
# Make config readable
chmod 600 my-config.yml
chown 1000:1000 my-config.yml

# Make logs directory writable
mkdir -p logs
chown 1000:1000 logs
```

---

## Networking

### Host Network Mode

For better performance with many proxy connections:

```bash
docker run --network host \
  -v $(pwd)/my-config.yml:/app/config/config.yml:ro \
  -e ADMIN_PASSWORD_BCRYPT_HASH="${ADMIN_PASSWORD_BCRYPT_HASH}" \
  -e JWT_SIGNING_SECRET_KEY="${JWT_SIGNING_SECRET_KEY}" \
  knock-knock-portal:latest
```

**Note:** Port mapping is ignored in host mode. Container uses host's network stack directly.

### Bridge Network (Default)

Recommended for isolated deployment:

```bash
docker network create knock-knock-net

docker run -d \
  --network knock-knock-net \
  -p 8000:8000 \
  -p 8080-8099:8080-8099 \
  knock-knock-portal:latest
```

---

## Monitoring & Logs

### View Logs

```bash
# Real-time logs
docker logs -f knock-knock-portal

# Last 100 lines
docker logs --tail 100 knock-knock-portal

# Since specific time
docker logs --since 1h knock-knock-portal
```

### Health Checks

```bash
# Manual health check
docker exec knock-knock-portal /app/knock-knock --health-check

# Docker health status
docker inspect --format='{{.State.Health.Status}}' knock-knock-portal
```

### Metrics (if using Prometheus)

```yaml
# Add to docker-compose.yml
services:
  knock-knock-portal:
    labels:
      - "prometheus.scrape=true"
      - "prometheus.port=8000"
      - "prometheus.path=/metrics"
```

---

## Troubleshooting

### Container Won't Start

```bash
# Check logs for errors
docker logs knock-knock-portal

# Common issues:
# - Missing environment variables
# - Invalid config.yml syntax
# - Port already in use
# - Permission issues with mounted volumes
```

### Test Container Interactively

```bash
# Run with shell access
docker run -it --rm \
  -v $(pwd)/my-config.yml:/app/config/config.yml:ro \
  knock-knock-portal:latest sh

# Inside container:
/app $ ls -la
/app $ cat config/config.yml
/app $ /app/knock-knock --version
```

### Check Binary Size

```bash
# Extract binary to inspect
docker create --name temp knock-knock-portal:latest
docker cp temp:/app/knock-knock ./knock-knock-binary
docker rm temp

# Check size
ls -lh knock-knock-binary

# Should be ~15-20MB (stripped binary)
```

### Verify Frontend Files

```bash
# Check frontend files are included
docker run --rm knock-knock-portal:latest ls -la /app/dist_frontend

# Should show:
# - index.html
# - _app/
# - favicon.png
# - etc.
```

---

## Security Best Practices

### 1. Use Secrets Management

**Never** hardcode secrets in Dockerfile or docker-compose.yml.

Use Docker secrets, environment variables from `.env`, or external secret managers (Vault, AWS Secrets Manager).

### 2. Run as Non-Root

Already configured in Dockerfile:
```dockerfile
USER knockknock  # UID 1000
```

### 3. Read-Only Root Filesystem

Add to `docker run`:
```bash
--read-only --tmpfs /tmp:noexec,nosuid,size=50M
```

### 4. Drop Capabilities

```bash
docker run --cap-drop=ALL ...
```

### 5. Use Security Scanning

```bash
# Scan image for vulnerabilities
docker scan knock-knock-portal:latest

# Or use Trivy
trivy image knock-knock-portal:latest
```

### 6. Network Segmentation

```bash
# Create isolated network
docker network create --internal backend-net

# Frontend-facing proxy
docker network create frontend-net

# Connect appropriately
docker run --network backend-net ...
```

---

## Performance Tuning

### Resource Limits

```yaml
services:
  knock-knock-portal:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M
```

### Connection Limits

Adjust in `config.yml`:
```yaml
proxy_server_config:
  max_connections_per_service: 2000  # Increase for high load
  connection_timeout_seconds: 60
```

### Kernel Parameters (Host)

For high connection counts:
```bash
# On host machine
sudo sysctl -w net.core.somaxconn=4096
sudo sysctl -w net.ipv4.ip_local_port_range="1024 65535"
sudo sysctl -w net.ipv4.tcp_max_syn_backlog=4096
```

---

## Image Size Comparison

```
Full Build Stages:
- frontend-builder (node:20-alpine): ~400MB
- backend-builder (golang:1.23-alpine): ~500MB
- Final image (alpine:3.19): ~30MB ✓

Size breakdown:
- Alpine base: ~7MB
- Go binary (stripped): ~15MB
- Frontend dist: ~5MB
- Dependencies: ~3MB
Total: ~30MB
```

---

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Build Docker Image

on:
  push:
    branches: [main]
    tags: ['v*']

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v2
      
      - name: Build and push
        uses: docker/build-push-action@v4
        with:
          context: .
          push: true
          tags: |
            ghcr.io/${{ github.repository }}:latest
            ghcr.io/${{ github.repository }}:${{ github.sha }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
```

---

## Next Steps

1. ✅ Build image
2. ✅ Test locally with docker-compose
3. ⏭️ Deploy to staging
4. ⏭️ Configure monitoring
5. ⏭️ Set up automated backups
6. ⏭️ Configure reverse proxy (nginx/Traefik)
7. ⏭️ Enable TLS/HTTPS
8. ⏭️ Deploy to production
