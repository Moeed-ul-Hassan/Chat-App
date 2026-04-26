# Echo Infrastructure & Deployment

This document outlines the infrastructure strategy for the Echo Backend, focusing on containerization, high availability, and cloud-native deployment.

## 1. Local Development Stack (Docker Compose)

For local development, Echo uses a multi-container setup to ensure environment parity.

```mermaid
graph TD
    subgraph "Local Sandbox"
        B[Go Backend:8001] -- "TCP" --> M[(MongoDB:27017)]
        B -- "TCP" --> R[(Redis:6379)]
    end
    
    User[Developer] -- "curl/Swagger" --> B
```

## 2. Production Architecture (GCP)

Echo is designed to be deployed as a serverless backend on **Google Cloud Platform (GCP)**.

```mermaid
graph LR
    subgraph "External"
        Internet[Public Internet]
    end

    subgraph "GCP Project"
        GCR[Cloud Run: Go Backend]
        M[(MongoDB Atlas: Shared Managed)]
        R[(Upstash/Redis: Managed)]
        
        DNS[Cloud DNS / Load Balancer] -- "HTTPS" --> GCR
        GCR -- "Atlas Driver" --> M
        GCR -- "Redis Driver" --> R
    end

    Internet -- "HTTPS/WSS" --> DNS
```

## 3. Deployment Pipeline (CI/CD)

The transition from code to production follows a standard Automated pipeline.

```mermaid
graph LR
    A[Git Push] --> B[GitHub Actions]
    B -- "Analyze" --> C[Go Test / Lint]
    C -- "Build" --> D[Docker Build & Tag]
    D -- "Store" --> E[Artifact Registry]
    E -- "Deploy" --> F[GCP Cloud Run Service]
```

## 4. Scalability Metrics

The backend is designed for horizontal scaling (Stateless).
- **Scale Out**: Cloud Run automatically scales from 0 to 100+ instances based on request CPU utilization.
- **WebSocket Scaling**: Currently handled via a single-instance Hub. Future scaling will involve **Redis Pub/Sub** to sync messages across multiple backend pods.
