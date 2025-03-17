
# SMPP USSD Gateway

This project implements a USSD Gateway using SMPP in Go. It supports:
- Multi-telco SMPP connections
- HTTP GET/POST for USSD menus
- Transaction logging to PostgreSQL
- REST API for reporting

## Setup

1. Configure `config/config.yaml` with your telco and route settings.
2. Setup PostgreSQL using the provided schema.
3. Build and run:

```bash
go build -o ussd_gateway ./cmd
./ussd_gateway
```

Access REST API at: `http://localhost:9080/api/reports/summary`
