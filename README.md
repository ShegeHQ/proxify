# Proxify

A lightweight reverse proxy with IP whitelisting for accessing private services in AWS and GCP VPCs.

## What It Does

Simple HTTP reverse proxy that forwards requests from whitelisted IPs to private backend services. Perfect for accessing databases, APIs, or staging environments in private subnets without VPNs or complex networking.

```
Your IP → Proxify (Public) → Private Service (VPC)
```

## Common Use Cases

**Access private RDS/Cloud SQL** - Connect to databases in private subnets from your local machine

**Expose internal APIs** - Give partners/services access to specific internal endpoints

**Dev/staging access** - Let developers reach staging environments without opening security groups to the world

**Cross-region/cloud bridge** - Connect services across VPCs or cloud providers

## Installation & Setup

### Prerequisites
- Go 1.23 or later
- Access to a VPC network (AWS VPC or GCP VPC)
- A virtual machine with network access to your target service

### Quick Start

1. **Clone and build:**
```bash
git clone <repository-url>
cd proxify
go build -o proxify server.go
```

2. **Configure environment variables:**
```bash
cp .env.example .env
# Edit .env with your settings
```

3. **Set your target and whitelist:**
```bash
# .env file
TARGET=http://your-private-service:8080
WHITELIST=203.0.113.0/24,198.51.100.45
```

4. **Run the proxy:**
```bash
./proxify
```

The proxy will start on port 8888 and forward requests to your target.

## Configuration

### Environment Variables

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `TARGET` | Yes | The backend service URL to proxy to | `http://10.0.1.50:3000` |
| `WHITELIST` | No | Comma-separated list of allowed IPs/CIDRs | `192.168.1.0/24,10.0.0.5` |

### Whitelist Format

The whitelist supports both individual IPs and CIDR notation:

```bash
# Individual IPs
WHITELIST=203.0.113.45,198.51.100.67

# CIDR ranges
WHITELIST=10.0.0.0/8,172.16.0.0/12

# Mixed
WHITELIST=203.0.113.0/24,198.51.100.45,192.168.1.100
```

If `WHITELIST` is not set, all IPs are allowed (not recommended for production).

## Deployment Examples

### AWS App Runner (Recommended for AWS)

The easiest way to deploy on AWS - fork this repo and deploy directly to App Runner.

1. **Fork this repository** to your GitHub account

2. **Create VPC Connector:**
   - Go to App Runner → VPC connectors → Create VPC connector
   - Select your VPC and private subnets where your target service lives
   - Select a security group that allows outbound access to your target

3. **Create App Runner Service:**
   - Go to App Runner → Services → Create service
   - Source: Source code repository → GitHub
   - Connect to GitHub and select your forked repo
   - Branch: `main`
   - Build settings:
     - Runtime: Go
     - Build command: `go build -o proxify server.go`
     - Start command: `./proxify`
     - Port: `8888`
   - Add environment variables:
     - `TARGET`: `http://10.0.1.50:3000` (your private service)
     - `WHITELIST`: `203.0.113.0/24` (your allowed IPs)
   - Networking: Select your VPC connector
   - Create service

4. App Runner provides an HTTPS URL like `https://xyz.region.awsapprunner.com`

**Benefits:** No server management, automatic scaling, built-in HTTPS, pay only for usage

### GCP Compute Engine

1. Create a Compute Engine instance with external IP
2. Configure firewall rules:
   - Allow ingress on port 8888 from whitelisted IPs
   - Allow egress to target service
3. Deploy Proxify

```bash
# Example: Access Cloud SQL via private IP
TARGET=http://10.128.0.50:3306
WHITELIST=198.51.100.0/24
```

### Docker Deployment

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o proxify server.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/proxify .
EXPOSE 8888
CMD ["./proxify"]
```

```bash
docker build -t proxify .
docker run -p 8888:8888 \
  -e TARGET=http://10.0.1.50:8080 \
  -e WHITELIST=203.0.113.0/24 \
  proxify
```

## Security Considerations

1. **Always use a whitelist in production** - Running without a whitelist exposes your backend to the internet
2. **Use HTTPS for sensitive data** - Consider placing Proxify behind a TLS-terminating load balancer
3. **Keep the whitelist minimal** - Only add IPs that absolutely need access
4. **Monitor logs** - Track access patterns and unauthorized attempts
5. **Network security** - Ensure your target service's security groups only allow traffic from the Proxify instance

## Monitoring

Proxify logs all requests in the format:
```
2024/10/17 12:34:56 GET /api/users
2024/10/17 12:34:57 POST /api/orders
```

Forbidden requests (non-whitelisted IPs) return HTTP 403 but are not logged by default.

## Alternatives & When to Use Them

- **AWS VPN / GCP Cloud VPN**: Better for multiple users needing access to many resources
- **Bastion Host**: Better for SSH/RDP access to instances
- **AWS PrivateLink / VPC Peering**: Better for high-volume, production-grade service-to-service communication
- **Proxify**: Best for simple, low-volume HTTP access with IP-based access control
