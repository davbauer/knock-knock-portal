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


## 🎯 Problem & Solution

**Problem**: You host game servers, development services, or APIs that you want to protect from bots and unauthorized access, but you don't want the complexity of a full VPN solution.

**Solution**: Knock-Knock Portal acts as a smart gateway that:
- 🔒 **Blocks all traffic by default** to protected ports
- 🌐 **Provides a web portal** where authorized users can login
- ⚡ **Dynamically whitelists** user IPs after authentication
- ⏱️ **Time-based access** with configurable session durations
- 🔄 **Auto-extends sessions** on active connections (optional)

## 🏗️ Architecture

```
┌─────────────┐
│   Internet  │
└──────┬──────┘
       │
┌──────▼──────────────────────────┐
│  Knock-Knock Portal (Docker)    │
│  ┌────────────────────────────┐ │
│  │  Web Portal (Svelte)       │ │  ← Users authenticate here
│  └────────────────────────────┘ │
│  ┌────────────────────────────┐ │
│  │  Go Backend & Proxy        │ │  ← Manages IP whitelist
│  │  - JWT Auth                │ │
│  │  - Session Management      │ │
│  │  - IP Allowlist            │ │
│  │  - TCP/UDP/HTTP Proxy      │ │
│  └────────────────────────────┘ │
└──────┬──────────────────────────┘
       │ Only authenticated IPs pass
┌──────▼──────────────────────────┐
│   Protected Services             │
│  - Minecraft Server              │
│  - Web Apps                      │
│  - APIs                          │
│  - Databases                     │
└──────────────────────────────────┘
```

## ✨ Features

### Authentication & Security
- 🔐 **Bcrypt password hashing** with timing-safe comparison
- 🎫 **JWT tokens** for stateless admin authentication
- 🔑 **Session-based access** for portal users
- 🚦 **Rate limiting** on login endpoints (10/min portal, 5/min admin)
- 🛡️ **IP spoofing prevention** with trusted proxy configuration

### Session Management
- ⏰ **Configurable duration** (default: 1 hour)
- 🔄 **Auto-extend on activity** (optional, up to max duration)
- 🧹 **Automatic cleanup** of expired sessions
- 👥 **Multi-user support** with per-user service restrictions

### Network & Proxy
- 🌐 **Dynamic DNS support** - Resolves hostnames to IPs periodically
- 📍 **CIDR range matching** for permanent IP allowlists
- 🔀 **Protocol support**: TCP, UDP, HTTP reverse proxy
- 📝 **HTTP header manipulation** (inject, override, remove headers)
- ⚡ **High-performance** concurrent connection handling

### Configuration
- 📄 **YAML configuration** with hot-reload support
- 🔧 **Environment variable** overrides for sensitive data
- ✅ **Comprehensive validation** on startup
- 📊 **Structured JSON logging** (zerolog)

## 🚀 Quick Start

### Prerequisites
- Docker & Docker Compose
- OR: Go 1.21+ (for local development)

### Option 1: Docker Compose (Recommended)

```bash
# 1. Clone the repository
git clone https://github.com/davbauer/knock-knock-portal.git
cd knock-knock-portal

# 2. Configure environment
cp backend/.env.example backend/.env
cp backend/config.example.yml backend/config.yml

# 3. Generate password hashes
cd backend
go run scripts/generate_hash.go "your-admin-password"
go run scripts/generate_hash.go "user-password"

# 4. Update .env and config.yml with generated hashes

# 5. Start the services
cd ..
docker-compose up -d

# 6. Access the portal
# Portal: http://localhost:8000
# API: http://localhost:8000/api/
```

### Option 2: Local Development

```bash
cd backend

# Install dependencies
go mod download

# Generate passwords
go run scripts/generate_hash.go "password123"

# Copy and configure
cp .env.example .env
cp config.example.yml config.yml
# Edit .env and config.yml

# Run
go run cmd/server/main.go
```

## 📖 Usage Example

### 1. User Logs In
```bash
curl -X POST http://localhost:8000/api/portal/login \
  -H "Content-Type: application/json" \
  -d '{"username":"steve","password":"secret123"}'

# Response:
{
  "message": "Login successful",
  "data": {
    "jwt_access_token": "eyJ...",
    "session_id": "uuid",
    "session_info": {
      "username": "steve",
      "authenticated_ip": "203.0.113.45",
      "expires_at": "2025-10-25T13:30:00Z",
      "auto_extend_enabled": true,
      "allowed_services": ["Minecraft Server", "Web API"]
    }
  }
}
```

### 2. User's IP Gets Whitelisted
The user's IP (`203.0.113.45`) is now allowed to access configured services on protected ports.

### 3. User Connects to Service
```bash
# User can now connect to Minecraft server
minecraft-connect minecraft.example.com:25565

# Their connection is proxied through, and session auto-extends
```

### 4. Admin Monitors Sessions
```bash
# Admin logs in
curl -X POST http://localhost:8000/api/admin/login \
  -H "Content-Type: application/json" \
  -d '{"admin_password":"admin-secret"}'

# View active sessions
curl -H "Authorization: Bearer <admin-token>" \
  http://localhost:8000/api/admin/sessions

# Terminate a session
curl -X DELETE \
  -H "Authorization: Bearer <admin-token>" \
  http://localhost:8000/api/admin/sessions/<session-id>
```

## ⚙️ Configuration

### Protected Services Example

```yaml
protected_services:
  - service_id: "uuid"
    service_name: "Minecraft Server"
    proxy_listen_port_start: 25565
    proxy_listen_port_end: 25565
    backend_target_host: "127.0.0.1"
    backend_target_port_start: 25565
    backend_target_port_end: 25565
    transport_protocol: "tcp"  # tcp | udp | both
    is_http_protocol: false
    enabled: true
    description: "Main game server"
```

### User Accounts Example

```yaml
portal_user_accounts:
  - user_id: "uuid"
    username: "steve"
    display_username_in_public_login_suggestions: true
    bcrypt_hashed_password: "$2a$10$..."
    allowed_service_ids: ["uuid1", "uuid2"]  # Empty = all services
    notes: "Steve's account"
```

## 🔧 Development Status

### ✅ Completed
- Configuration system with hot-reload
- Authentication (JWT, bcrypt, rate limiting)
- Session management with auto-extend
- IP allowlist with DNS resolution
- RESTful API (portal + admin endpoints)
- Health checks and structured logging

### 🚧 To Do
- TCP/UDP/HTTP proxy implementation
- Frontend (Svelte web portal)
- Comprehensive test suite
- Prometheus metrics endpoint
- GeoIP blocking (optional)
- 2FA support (optional)

## 📚 API Documentation

See [backend/README.md](backend/README.md) for complete API documentation.

## 🛠️ Technology Stack

**Backend:**
- Go 1.21+
- Gin (HTTP framework)
- JWT (golang-jwt/jwt)
- Zerolog (structured logging)
- Bcrypt (password hashing)
- YAML (configuration)

**Frontend** (planned):
- Svelte
- TypeScript
- TailwindCSS

## 🤝 Contributing

Contributions welcome! This is currently in active development.

## 📄 License

MIT License - see LICENSE file for details

## 🎯 Use Cases

Perfect for:
- 🎮 **Game servers** (Minecraft, Valheim, etc.)
- 🌐 **Development services** (staging environments)
- 🔒 **Internal APIs** (protected but accessible)
- 📊 **Self-hosted applications** (Grafana, Portainer, etc.)
- 🏠 **Home lab services** with dynamic IP

---

**Demo Credentials:**
- Username: `demo`
- Password: `password123`

⚠️ **Change these in production!**
