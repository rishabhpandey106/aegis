# Frequently Asked Questions (FAQ)

Welcome to the Aegis Firewall FAQ. This document covers common architectural, operational, and failover questions regarding the Aegis platform.

## Architecture & Failover

### What happens if the Data Plane (Reverse Proxy) crashes?
The Data Plane Proxy is the only component in the direct critical path of your upstream API traffic. 
- **Impact:** If the Proxy goes down, all traffic routed through it will fail to reach your backend servers. 
- **Mitigation:** In production, you should deploy multiple instances of the Proxy behind a highly available Load Balancer (e.g., AWS ALB or NGINX). Because the Proxy is stateless (relying on PostgreSQL and Redis), it can scale horizontally to infinite instances.

### What happens if the Control Plane (API) goes down?
The Control Plane serves the Frontend Dashboard and handles user authentication, organization management, and configuration changes.
- **Impact:** The Dashboard will go offline, and administrators will be unable to log in, create new projects, or change security rules.
- **Proxy Resilience:** The Data Plane Proxy **will continue to function perfectly**. The proxy caches routing configurations and security rules in its memory. Even when the cache expires, it reads directly from PostgreSQL, meaning the proxy does not rely on the Control Plane to process live traffic!

### What happens if the Python AI Engine crashes?
The AI Engine evaluates payloads for zero-day threats using Machine Learning via a gRPC connection.
- **Impact:** The system is designed to **"Fail Open"** for the AI component. If the Proxy cannot reach the AI Engine via gRPC, it will instantly log a warning and allow the traffic to pass through rather than blocking legitimate requests.
- **Security:** Standard deterministic WAF rules (RegEx), Rate Limiting, and DLP will still continue to protect your upstream services perfectly. Only the advanced ML heuristic detection will be temporarily disabled.

### What happens if NATS (Message Broker) or the Analytics Worker goes down?
The Analytics Worker subscribes to NATS to ingest firehose telemetry logs from the Proxy.
- **Impact:** The Proxy is designed to **"Fail Open"** for telemetry. If NATS is down, the Proxy will simply drop the telemetry logs and continue routing traffic. Your API users will experience zero downtime and zero latency penalties.
- **Data Loss:** You will temporarily lose visibility and traffic metrics on the Dashboard until NATS and the Analytics Worker are restored, but live traffic will flow uninterrupted.

### What happens if Redis crashes?
Redis is used exclusively for Rate Limiting.
- **Impact:** The Rate Limiting middleware will fail. The Proxy is designed to bypass the rate limiter if Redis is unreachable, allowing traffic to flow so your business doesn't go offline.
- **Security:** You may be temporarily vulnerable to volumetric DDoS attacks or API scraping until Redis is restored.

### What happens if PostgreSQL goes down?
PostgreSQL is the ultimate source of truth for Organizations, Projects, Users, and Security Rules.
- **Impact:** The Control Plane will fail, bringing down the Dashboard. 
- **Proxy Resilience:** The Proxy uses an In-Memory Cache for 5 minutes. If Postgres goes down, the Proxy will continue routing traffic using its cached rules for up to 5 minutes. After the cache expires, if Postgres is still down, the Proxy will begin failing to route new traffic because it cannot determine where to send it.

---

## General Usage

### Can multiple organizations share the same email address?
Currently, our database enforces a strict globally unique constraint on user emails. This means a single email address can only belong to one Organization at a time. To support multi-tenant accounts in the future, the database schema would need to introduce a many-to-many `organization_members` table.

### How does Clerk integrate with Aegis?
Aegis enforces a Zero-Trust policy. When a user logs into the Next.js Dashboard via Clerk, Clerk issues a cryptographically signed JWT. The Go Control Plane intercepts every API request, downloads Clerk's public JWKS keys, and verifies the JWT signature locally before granting access to resources.

## Architectural Decisions

### Why gRPC instead of REST for the AI Engine?
The Data Plane Proxy must query the Python AI Engine on every incoming HTTP request. Using standard REST (JSON over HTTP/1.1) introduces massive serialization overhead and network latency. **gRPC** uses Protobufs over HTTP/2, which is highly compressed, binary-serialized, and uses persistent multiplexed connections. This allows the Proxy to evaluate ML threats in sub-millisecond times, which is impossible with REST.

### Why NATS instead of Kafka?
NATS is chosen over Kafka for its extreme simplicity, lightweight footprint, and incredibly low latency. Kafka requires significant operational overhead (JVM, Zookeeper/KRaft, persistent disk management) which is overkill for ephemeral firehose telemetry. NATS allows the Proxy to dump millions of logs per second asynchronously without blocking the main HTTP thread.

### Why use both Redis and an In-Memory Cache?
- **In-Memory Cache (RAM):** Used strictly for routing configuration (Projects, WAF rules, API keys). Reading from RAM takes less than 1 microsecond, avoiding any network hops entirely.
- **Redis:** Used exclusively for **Rate Limiting**. Rate limiting *must* be distributed. If you scale the Proxy to 10 instances, an attacker could bypass an in-memory rate limiter just by hitting different instances. Redis ensures a global, centralized count of requests across all proxies.

### What is the Fallback Mechanism?
Aegis is built around the **"Fail Open"** philosophy. If a non-critical component (AI Engine, NATS Analytics, or Redis) goes offline, the Proxy will instantly catch the error, log a warning, and allow the traffic to proceed to the upstream server. The primary goal of the Proxy is to ensure your business APIs stay online, even if some security heuristics are temporarily degraded.

### What Rate Limiter algorithm is used?
Aegis implements a highly efficient **Fixed Window** algorithm using Redis Pipelines. It calculates the current time window, executes an atomic `INCR` and `EXPIRE` command in a single network roundtrip, and checks if the threshold is exceeded.

### How are False Positives handled?
False positives in the AI Engine can be mitigated by adjusting the confidence threshold in the security rule configuration. If a specific ML rule is triggering too aggressively on legitimate traffic, an administrator can instantly disable the AI Blocker rule for that specific project from the Dashboard, falling back to deterministic RegEx WAF rules.

### How does the Circuit Breaker work?
Aegis implements a **Half-Open Circuit Breaker** using the `sony/gobreaker` library. 
If your upstream API server starts crashing or timing out, the Circuit Breaker trips to the **Open** state. While Open, Aegis instantly blocks all incoming traffic and returns a `503 Service Unavailable`, preventing your backend from being overwhelmed while it tries to recover. After a cooldown period, it enters a **Half-Open** state, letting a few test requests through. If they succeed, the circuit closes and traffic resumes normally.
