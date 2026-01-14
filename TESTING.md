# Local Testing Environment - Getting Started

Complete Docker-based testing environment for the WhatsApp Recovery System.

## Prerequisites

- Docker Desktop installed and running
- Docker Compose installed (included with Docker Desktop)
- Git Bash (Windows) or Terminal (Mac/Linux)

## Quick Start

### 1. Navigate to the docker directory

```bash
cd docker
```

### 2. Make scripts executable (Linux/Mac/Git Bash)

```bash
chmod +x ../local/scripts/*.sh
```

### 3. Run the setup script

```bash
../local/scripts/setup.sh
```

This will:
- Build all Docker images
- Start all services
- Wait for health checks
- Display access information

### 4. Verify services are running

```bash
docker-compose ps
```

All services should show status "Up" and healthy.

## Test the System

### Post a journey event

```bash
curl -X POST http://localhost:8080/journey/event \
  -H "Content-Type: application/json" \
  -d '{
    "journey_id": "onboarding-v2",
    "step": "personal-data",
    "customer_number": "5511999999999",
    "tenant_id": "tenant-123",
    "contact_id": "contact-456",
    "metadata": {
      "nome_cliente": "João Silva",
      "link": "https://example.com/continue"
    }
  }'
```

### Check Redis state

```bash
docker-compose exec redis redis-cli

> KEYS journey:*
> GET journey:onboarding-v2:5511999999999:state
> exit
```

### View worker logs

```bash
docker-compose logs -f worker
```

Wait for the worker to execute (every 5 minutes). You should see:
- "Starting scheduled worker execution"
- "Loaded journey config"
- Eventually (after 10 minutes of inactivity): "MOCK: Would send WhatsApp message"

### Stop the environment

```bash
docker-compose down
```

## What's Included

✅ **Event Tracker API** - HTTP API on port 8080
✅ **Worker Service** - Runs every 5 minutes to check for abandoned journeys
✅ **Redis** - Stores journey states and recovery history
✅ **AppConfig Mock** - Serves journey configurations from local files
✅ **WhatsApp Mock** - Logs messages instead of sending them
✅ **Helper Scripts** - For testing and inspecting data

## Next Steps

See [local/README.md](local/README.md) for:
- Detailed testing workflows
- Helper script usage
- Troubleshooting guide
- Configuration hot-reload
- Advanced testing scenarios

## Windows Users

If you're using Windows without Git Bash, you can run Docker Compose commands directly:

```powershell
cd docker
docker-compose build
docker-compose up -d
docker-compose logs -f
```

For the curl commands, you can use PowerShell's `Invoke-RestMethod`:

```powershell
$body = @{
    journey_id = "onboarding-v2"
    step = "personal-data"
    customer_number = "5511999999999"
    tenant_id = "tenant-123"
    contact_id = "contact-456"
    metadata = @{
        nome_cliente = "João Silva"
        link = "https://example.com/continue"
    }
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8080/journey/event" -Method Post -Body $body -ContentType "application/json"
```

## Support

If you encounter issues:
1. Check Docker is running: `docker ps`
2. View service logs: `docker-compose logs [service-name]`
3. Verify health: `docker-compose ps`
4. Review [local/README.md](local/README.md) for troubleshooting
