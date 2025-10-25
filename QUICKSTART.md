# Quick Start Guide

## Get Running in 2 Minutes

### 1. Prerequisites
- Go 1.21+ installed
- `curl` for testing

### 2. Setup

```bash
# Navigate to backend
cd backend

# Copy example files
cp .env.example .env
cp config.example.yml config.yml

# Install dependencies
go mod download
```

### 3. Run

```bash
# Start the server
go run cmd/server/main.go
```

You should see:
```json
{"level":"info","version":"1.0.0","message":"Starting Knock-Knock Portal"}
{"level":"info","message":"Configuration loaded successfully"}
{"level":"info","port":8000,"message":"HTTP API server started"}
```

### 4. Test

Open a new terminal:

```bash
# Check health
curl http://localhost:8000/health

# Login
curl -X POST http://localhost:8000/api/portal/login \
  -H "Content-Type: application/json" \
  -d '{"username":"demo","password":"password123"}' \
  | python3 -m json.tool
```

Expected response:
```json
{
  "message": "Login successful",
  "data": {
    "jwt_access_token": "eyJ...",
    "session_id": "...",
    "session_info": {
      "username": "demo",
      "authenticated_ip": "::1",
      "expires_at": "...",
      "auto_extend_enabled": true,
      "allowed_services": ["Example Service"]
    }
  }
}
```

## What Now?

### Change Passwords

```bash
# Generate new hash
go run scripts/generate_hash.go "my-secure-password"

# Copy the hash and update:
# - .env for admin password
# - config.yml for portal users
```

### Add More Users

Edit `config.yml`:

```yaml
portal_user_accounts:
  - user_id: "550e8400-e29b-41d4-a716-446655440000"
    username: "alice"
    display_username_in_public_login_suggestions: true
    bcrypt_hashed_password: "$2a$10$..."
    allowed_service_ids: []
    notes: "Alice's account"
```

Hot-reload will pick up changes automatically!

### Configure Services

Edit `config.yml`:

```yaml
protected_services:
  - service_id: "7c9e6679-7425-40de-944b-e07fc1f90ae7"
    service_name: "Minecraft Server"
    proxy_listen_port_start: 25565
    proxy_listen_port_end: 25565
    backend_target_host: "127.0.0.1"
    backend_target_port_start: 25565
    backend_target_port_end: 25565
    transport_protocol: "tcp"
    is_http_protocol: false
    enabled: true
```

⚠️ **Note:** Proxy functionality not yet implemented. This configures what will be proxied when the proxy system is built.

### Run with Docker

```bash
# Build
docker build -t knock-knock-portal .

# Run
docker run -p 8000:8000 \
  -v $(pwd)/config.yml:/app/config.yml \
  -v $(pwd)/.env:/app/.env \
  knock-knock-portal
```

## API Cheat Sheet

```bash
# Portal Login
POST /api/portal/login
{"username": "demo", "password": "password123"}

# Get Suggested Usernames
GET /api/portal/suggested-usernames

# Session Status (requires JWT)
GET /api/portal/session/status
Header: Authorization: Bearer <token>

# Logout (requires JWT)
POST /api/portal/session/logout
Header: Authorization: Bearer <token>

# Admin Login
POST /api/admin/login
{"admin_password": "password123"}

# List Sessions (requires admin JWT)
GET /api/admin/sessions
Header: Authorization: Bearer <admin-token>

# Terminate Session (requires admin JWT)
DELETE /api/admin/sessions/:session_id
Header: Authorization: Bearer <admin-token>

# Health Check
GET /health
```

## Troubleshooting

**Server won't start:**
- Check `.env` file exists and has valid password hash
- Check `config.yml` is valid YAML
- Check port 8000 is not already in use

**Login fails:**
- Verify password hash matches what you generated
- Check username exists in config.yml
- Check rate limiting (max 10/min per IP)

**Configuration not loading:**
- Check YAML syntax is valid
- Check file permissions
- Check logs for validation errors

## Next Steps

- Read [ARCHITECTURE.md](ARCHITECTURE.md) for system design
- Read [backend/README.md](backend/README.md) for detailed docs
- See [PROJECT_STATUS.md](PROJECT_STATUS.md) for what's completed
- Customize config.yml for your use case

---

**That's it! You're running Knock-Knock Portal!** 🎉
