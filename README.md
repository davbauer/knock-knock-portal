# Knock-Knock Portal

Secure authentication gateway with session-based IP allowlisting for your services.

## Setup

1. **Get the code**
   ```bash
   git clone https://github.com/davbauer/knock-knock-portal.git
   cd knock-knock-portal
   ```

2. **Create secrets**
   ```bash
   # Generate admin password hash
   htpasswd -bnBC 12 "" yourpassword | tr -d ':\n'
   
   # Generate JWT secret
   openssl rand -base64 32
   ```

3. **Configure**
   ```bash
   cp .env.example .env
   # Add your secrets to .env
   # Edit backend/config.yml for your services
   ```

4. **Run**
   ```bash
   docker-compose up -d
   ```

Access portal at `http://localhost:8000`

## Docker Images

Pre-built images available at GitHub Container Registry:

```bash
# AMD64
docker pull ghcr.io/davbauer/knock-knock-portal:main-amd64

# ARM64
docker pull ghcr.io/davbauer/knock-knock-portal:main-arm64
```

Use in [docker-compose.yml](docker-compose.yml):
```yaml
services:
  knock-knock-portal:
    image: ghcr.io/davbauer/knock-knock-portal:main-amd64
    # ... rest of config
```

## Development

```bash
# Frontend
cd frontend && yarn dev

# Backend
cd backend && go run cmd/server/main.go
```

## License

MIT
