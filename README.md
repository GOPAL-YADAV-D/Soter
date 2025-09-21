# Soter - Secure File Vault System

[![CI/CD Pipeline](https://github.com/GOPAL-YADAV-D/Soter/actions/workflows/ci-cd.yml/badge.svg)](https://github.com/GOPAL-YADAV-D/Soter/actions/workflows/ci-cd.yml)
[![Security Scan](https://github.com/GOPAL-YADAV-D/Soter/actions/workflows/security.yml/badge.svg)](https://github.com/GOPAL-YADAV-D/Soter/actions/workflows/security.yml)
[![codecov](https://codecov.io/gh/GOPAL-YADAV-D/Soter/branch/main/graph/badge.svg)](https://codecov.io/gh/GOPAL-YADAV-D/Soter)

A production-grade secure file storage and sharing system with enterprise-level security, rate limiting, audit logging, virus scanning, and Azure Blob Storage integration.

## 🆕 Latest Enhancements

### 🔒 Enterprise Security Features
- **CSRF Protection**: Double-submit cookie pattern with secure headers
- **Rate Limiting**: Token bucket algorithm with per-user/organization limits
- **Audit Logging**: Comprehensive security event tracking
- **Virus Scanning**: ClamAV integration for malware detection
- **Azure Storage**: Secure file storage with SAS URL downloads

### 📊 Monitoring & Performance
- **Redis Caching**: Distributed rate limiting and session storage
- **Prometheus Metrics**: Real-time performance monitoring
- **Grafana Dashboards**: Visual analytics and alerting
- **Health Checks**: Service availability monitoring

## 🧪 Testing

📋 **[Complete Testing Guide](TESTING_GUIDE.md)** - Comprehensive testing procedures for all features

Quick start testing:
```bash
./scripts/setup-testing.sh
docker-compose up -d
./scripts/test-features.sh
```

## 🏗️ Architecture Overview

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   React + TS    │    │    Go Backend    │    │   PostgreSQL    │
│   Frontend      │◄──►│   GraphQL API    │◄──►│   Database      │
│   (Port 3000)   │    │   (Port 8080)    │    │   (Port 5432)   │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       │
         │              ┌────────┼────────┐             │
         │              │                 │             │
         ▼              ▼                 ▼             ▼
┌─────────────────┐  ┌─────────────────┐ ┌─────────────────┐
│    Nginx        │  │    Azurite      │ │   Prometheus    │
│   Reverse       │  │  Blob Storage   │ │   + Grafana     │
│   Proxy         │  │  (Port 10000)   │ │  (Ports 9090,   │
└─────────────────┘  └─────────────────┘ │   3001)         │
                                         └─────────────────┘
```

## ✨ Features

### 🔒 **Core Security**
- JWT-based authentication with middleware
- Argon2id password hashing
- HTTPS/TLS enforcement
- CSRF protection
- Linux-style permissions (Owner/Group/Others)
- Role-based access control (RBAC)
- Audit logging for compliance

### 📁 **File Management**
- **Deduplication**: SHA-256 based with reference counting
- **Multi-upload**: Single and batch file uploads
- **Chunked uploads**: For files >100MB
- **MIME validation**: Real vs declared type checking
- **Secure sharing**: Public links with expiry, private access, per-user/org permissions
- **Folder hierarchy**: Organized file structure
- **Download tracking**: Public file analytics

### 🔍 **Search & Discovery**
- Full-text search by filename
- Advanced filtering: MIME type, size, date, tags, uploader
- PostgreSQL optimized with indexes and full-text search
- Combinable filter system

### 📊 **Analytics & Monitoring**
- **Storage stats**: Logical vs physical storage tracking
- **Deduplication insights**: Bytes and percentage saved
- **Admin dashboard**: Global usage metrics and charts
- **Real-time monitoring**: Prometheus metrics + Grafana dashboards
- **Structured logging**: JSON with request tracing
- **Health checks**: Database and storage connectivity

### ⚡ **Performance & Scalability**
- **Rate limiting**: Token bucket (2 req/s per user, configurable)
- **Storage quotas**: 10MB per user (configurable)
- **Connection pooling**: Optimized database connections
- **Caching strategies**: Built-in optimization
- **Container orchestration**: Docker Compose ready

### 🏢 **Multi-tenancy**
- **Organizations**: Users can belong to multiple orgs
- **Admin controls**: Org creation, user management
- **Invitation system**: Secure user onboarding
- **Resource isolation**: Per-org quotas and permissions

## 🚀 Quick Start

### Prerequisites
- **Docker** & **Docker Compose** (20.10+)
- **Go** 1.21+ (for local development)
- **Node.js** 18+ (for frontend development)
- **Git**

### 1. Clone Repository
```bash
git clone https://github.com/GOPAL-YADAV-D/Soter.git
cd Soter
```

### 2. Start with Docker Compose
```bash
# Start all services
docker-compose up -d

# Check service health
docker-compose ps

# View logs
docker-compose logs -f
```

### 3. Access the Application
- **Frontend**: http://localhost:3000
- **GraphQL Playground**: http://localhost:8080/playground
- **Health Check**: http://localhost:8080/healthz
- **Metrics**: http://localhost:8080/metrics
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3001 (admin/admin123)

### 4. Default Credentials
- **Admin User**: `admin@soter.local` / `admin123`
- **Grafana**: `admin` / `admin123`

## 🛠️ Development Setup

### Backend Development
```bash
cd backend

# Install dependencies
go mod download

# Generate GraphQL code
go run github.com/99designs/gqlgen generate

# Run tests
go test -v ./...

# Run locally (requires running database)
go run cmd/server/main.go
```

### Frontend Development
```bash
cd frontend

# Install dependencies
npm install

# Start development server
npm start

# Run tests
npm test

# Build for production
npm run build
```

### Database Management
```bash
# Connect to PostgreSQL
docker-compose exec postgres psql -U postgres -d soter

# View database logs
docker-compose logs postgres

# Reset database
docker-compose down -v
docker-compose up -d postgres
```

## 📊 Monitoring & Observability

### Health Monitoring
The system provides comprehensive health checks:
- **Database connectivity**: PostgreSQL connection status
- **Storage backend**: Azure Blob/Azurite availability
- **Service dependencies**: Real-time dependency monitoring

### Metrics Collection
Prometheus metrics include:
- HTTP request rates and latency
- File upload/download statistics
- Database query performance
- Storage usage and deduplication efficiency
- Rate limiting violations
- Active user counts

### Logging Strategy
Structured JSON logging with:
- Request IDs for distributed tracing
- User context and actions
- Error tracking and alerting
- Audit trail for compliance

## 🔧 Configuration

### Environment Variables

#### Backend Configuration
```bash
# Server
PORT=8080
HOST=0.0.0.0

# Database
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=soter
DB_SSLMODE=disable

# Azure Storage
AZURE_STORAGE_ACCOUNT=devstoreaccount1
AZURE_STORAGE_KEY=<key>
AZURE_STORAGE_CONTAINER=files
AZURE_STORAGE_ENDPOINT=http://azurite:10000/devstoreaccount1

# Security
JWT_SECRET=your-super-secret-key
LOG_LEVEL=info

# Rate Limiting
RATE_LIMIT_RPS=2
RATE_LIMIT_BURST=5
STORAGE_QUOTA_MB=10
```

#### Frontend Configuration
```bash
REACT_APP_API_URL=http://localhost:8080
REACT_APP_GRAPHQL_URL=http://localhost:8080/query
```

## 🧪 Testing

### Running Tests
```bash
# Backend tests
cd backend
go test -v -race -coverprofile=coverage.out ./...

# Frontend tests
cd frontend
npm test -- --coverage

# Integration tests
docker-compose -f docker-compose.test.yml up --abort-on-container-exit
```

### Test Coverage
- Backend: Unit tests, integration tests, database tests
- Frontend: Component tests, API integration tests
- E2E: Full workflow testing with Docker Compose

## 🚀 Deployment

### Production Deployment

#### Using Docker Images
```bash
# Pull latest images
docker pull ghcr.io/gopal-yadav-d/soter/backend:latest
docker pull ghcr.io/gopal-yadav-d/soter/frontend:latest

# Deploy with custom configuration
docker-compose -f docker-compose.prod.yml up -d
```

#### Environment Setup
1. **Database**: Set up managed PostgreSQL (AWS RDS, Azure Database, etc.)
2. **Storage**: Configure Azure Blob Storage
3. **Monitoring**: Set up Prometheus + Grafana
4. **Security**: Configure HTTPS, environment secrets
5. **Scaling**: Use container orchestration (Kubernetes, Docker Swarm)

### Security Considerations
- Change default passwords and JWT secrets
- Enable HTTPS with proper certificates
- Configure firewall rules
- Set up backup strategies
- Enable audit logging
- Configure monitoring alerts

## 📊 API Documentation

### GraphQL Schema
The system uses GraphQL for the primary API. Access the interactive playground at `/playground` to explore:

#### Key Queries
- `health`: System health status
- `files`: List user files with filtering
- `fileStats`: Storage statistics and deduplication info
- `organizations`: User organizations and permissions

#### Key Mutations
- `uploadFile`: File upload with metadata
- `createOrganization`: Organization management
- `createUser`: User management

### REST Endpoints
- `GET /healthz`: Health check
- `GET /metrics`: Prometheus metrics
- `GET /api/v1/*`: RESTful API endpoints

## 🤝 Contributing

### Development Workflow
1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Commit changes: `git commit -m 'Add amazing feature'`
4. Push to branch: `git push origin feature/amazing-feature`
5. Open a Pull Request

### Code Standards
- **Go**: Follow `gofmt`, `golint`, and `go vet` standards
- **TypeScript**: Follow ESLint and Prettier configurations
- **Commits**: Use conventional commit messages
- **Tests**: Maintain >80% code coverage
- **Documentation**: Update README and inline docs

## 📋 Roadmap

### Current (v1.0) - ✅ Completed
- [x] Basic file upload/download
- [x] SHA-256 deduplication
- [x] GraphQL API
- [x] Docker containerization
- [x] Health monitoring
- [x] Basic security (JWT, RBAC)

### Next Release (v1.1)
- [ ] Virus scanning (ClamAV integration)
- [ ] Advanced search with Elasticsearch
- [ ] File versioning
- [ ] Bulk operations
- [ ] API rate limiting enhancements

### Future (v2.0)
- [ ] Distributed storage backends
- [ ] WebRTC direct transfers
- [ ] Advanced analytics dashboard
- [ ] Mobile app support
- [ ] Plugin architecture

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support & Community

- **Issues**: [GitHub Issues](https://github.com/GOPAL-YADAV-D/Soter/issues)
- **Discussions**: [GitHub Discussions](https://github.com/GOPAL-YADAV-D/Soter/discussions)
- **Security**: security@soter.local

## 🙏 Acknowledgments

- **PostgreSQL**: Robust database foundation
- **GraphQL**: Modern API development
- **Prometheus**: Metrics and monitoring
- **Docker**: Containerization platform
- **Azure**: Cloud storage integration

---

**Soter** - *Ancient Greek: Σωτήρ, meaning "savior" or "protector"*

Built with ❤️ for secure, scalable file management.