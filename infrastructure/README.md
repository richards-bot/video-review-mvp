# Infrastructure

This directory contains deployment configurations, infrastructure-as-code, and operational tooling.

## Structure

```
infrastructure/
├── docker/              # Container configurations
│   ├── Dockerfile
│   └── docker-compose.yml
├── terraform/           # Infrastructure as Code
│   ├── main.tf
│   ├── variables.tf
│   └── environments/
│       ├── dev/
│       ├── staging/
│       └── production/
├── kubernetes/          # K8s manifests (if applicable)
│   ├── deployment.yaml
│   └── service.yaml
├── scripts/             # Operational scripts
│   ├── deploy.sh
│   └── backup.sh
└── monitoring/          # Observability configs
    ├── alerts/
    └── dashboards/
```

## Environments

| Environment | Purpose | Access |
|-------------|---------|--------|
| Development | Local development | Developers |
| Staging | Pre-production testing | Team |
| Production | Live system | Restricted |

## Common Tasks

### Local Development
```bash
# Start local services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

### Deployment
```bash
# Deploy to staging
./scripts/deploy.sh staging

# Deploy to production
./scripts/deploy.sh production
```

### Infrastructure Changes
```bash
cd terraform/environments/staging

# Plan changes
terraform plan

# Apply changes
terraform apply
```

## Security Notes

### Secrets Management
- Never commit secrets to git
- Use environment variables or secret managers
- Rotate credentials regularly

### Access Control
- Production access is restricted
- All changes require review
- Audit logs are maintained

## For AI Agents

### Caution Areas
- **Do not modify** without explicit request
- **Do not commit** secrets or credentials
- **Always plan before apply** for Terraform
- **Test in staging** before production

### Safe Operations
- Reading configurations
- Explaining infrastructure
- Suggesting improvements
- Writing documentation

### Requires Human Approval
- Any production changes
- Security configuration changes
- Network/firewall changes
- Database migrations
