# Docker & Deployment Mastery — Part 2: Dev Compose, Production Topology, CI/CD

## Docker Compose as a Local Infrastructure Manager

Docker Compose is not a production deployment tool in this architecture. It is a local developer infrastructure manager. The distinction is important and the reason the architecture uses it the way it does.

In production, this project runs the Go binary as a raw systemd service. Redis runs as a native system service installed via `apt`. There is no Docker on the production Droplet at all. The reason is that for a single service talking to two infrastructure dependencies (Redis and Postgres), the overhead of the Docker daemon, container networking, and volume management adds complexity with zero operational benefit. The Go binary is already a single statically-linked file. Copying it over SSH and restarting a systemd unit is faster, simpler, and has fewer moving parts than maintaining a Docker registry, pulling images, and managing container lifecycle on the server.

Compose is used locally because you do not want to install Redis on your development machine as a system service. You want to start and stop the dev infrastructure in a single command, have it isolated from other projects, and have it gone when you are done.

Read the dev compose file with this mental model:

```yaml
services:
  redis:
    image: redis:7-alpine
    restart: unless-stopped
    command: redis-server --requirepass ${REDIS_PASSWORD:-devpassword} --bind 0.0.0.0
    ports:
      - "127.0.0.1:6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "-a", "${REDIS_PASSWORD:-devpassword}", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5
```

`image: redis:7-alpine` — same principle as the Go builder stage. Alpine reduces the image from 120MB to 32MB. For a Redis instance that exists only to serve your local development, this matters only in initial pull time, but the habit of reaching for Alpine images first is correct.

`restart: unless-stopped` — if the Redis container crashes (it almost never does), Compose restarts it. The exception is if you explicitly run `docker compose down`, which sets the desired state to stopped. This is different from `always` which would restart even after a deliberate stop.

`command: redis-server --requirepass ... --bind 0.0.0.0` — this overrides the default Redis entrypoint command. In dev, you bind to `0.0.0.0` inside the container (all container-internal interfaces), and then the `ports` mapping punches a hole from `127.0.0.1:6379` on your host to `6379` inside the container. The `${REDIS_PASSWORD:-devpassword}` syntax is shell parameter expansion: use the environment variable `REDIS_PASSWORD` if set, otherwise use the literal string `devpassword`. This means your `config/dev.env` can set `REDIS_URL=redis://:devpassword@127.0.0.1:6379/0` and it will work without any additional setup.

`ports: - "127.0.0.1:6379:6379"` — this is the port binding. The format is `[host_ip:]host_port:container_port`. By binding to `127.0.0.1` (loopback only), you ensure Redis is not accessible from outside your machine. This mirrors the production setup where Redis is bound to `127.0.0.1` via `redis.conf`. The habit of always binding dev services to loopback unless you have a specific reason not to is correct security hygiene.

`volumes: - redis_data:/data` — named volume. Docker manages this volume as a named object in its storage system. The data inside persists across `docker compose down && docker compose up` cycles. If you want to wipe the Redis state (simulate a fresh start), you run `docker compose down -v` — the `-v` flag removes named volumes along with containers.

`healthcheck` — Compose waits for `redis-cli ping` to return `PONG` before marking the service healthy. Other services that declare `depends_on: redis: condition: service_healthy` will not start until this passes. You do not use this in the current compose because the Go binary runs on the host, not in Compose. But when you later run everything in Compose (see the future scope section), this is the mechanism that prevents your app container from starting before Redis is ready.

## The `volumes:` top-level key

```yaml
volumes:
  redis_data:
```

This declares the named volume at the Compose project level. Without this declaration, the volume reference in the service definition would be treated as a bind mount (path on host) rather than a named volume (managed by Docker). A named volume survives `docker compose down` but not `docker compose down -v`.

## Running the Dev Workflow

The sequence for local development on this project is:

```
make infra-up     # starts Redis container
make dev          # builds and runs Go binary on host
make infra-down   # stops and removes Redis container (data persists in named volume)
```

The Go binary reads `config/dev.env` by default (see `configs/config.go`). Your dev.env has `REDIS_URL=redis://:devpassword@127.0.0.1:6379/0`. The binary connects to Redis running in the container through the port binding. From the binary's perspective, Redis is just something on `127.0.0.1:6379` — it does not know or care that Redis is inside a container.

---

## The Production Topology — Why No Docker in Production

The production machine is a single DigitalOcean Droplet running:

The Nginx process, managed by systemd, handling TLS termination and proxying. The Redis server process, managed by systemd, bound strictly to `127.0.0.1`. The Go binary process, managed by systemd, listening on `127.0.0.1:8080`. Nothing else.

There is no Docker daemon on this machine. The Go binary was cross-compiled on the CI runner and copied via SCP. This architecture is called a "bare binary" or "native service" deployment, and it is the correct choice at this scale for these reasons:

The Go binary is statically linked. It has no runtime dependencies. Installing Docker to run a binary that has no runtime dependencies is adding a daemon, a networking stack, a storage driver, a container runtime, and a namespace management layer to solve a problem that does not exist. The operational cost is real: the Docker daemon consumes ~100MB RAM, adds startup latency, and introduces a failure mode (daemon crash) that does not exist in the native approach.

Systemd does everything Docker's restart policy does, plus it integrates with journald for log aggregation, supports `EnvironmentFile` for secrets injection, handles `LimitNOFILE` and `LimitNPROC` kernel limits, manages graceful shutdown via `TimeoutStopSec` and `KillSignal`, and has been production-hardened for two decades. There is no reason to add Docker to the prod server if you are running a single service.

This is an important architectural decision to internalize: Docker is a packaging and isolation tool. Its appropriate use in production is when you have multiple services that would conflict (e.g., two services needing different versions of Python), when you need hermetic builds that must be identical across environments, or when you are deploying to a managed container platform. For a single statically-linked Go binary on a dedicated VM, systemd is strictly superior.

---

## The Systemd Unit — Full Understanding

```ini
[Unit]
Description=Airstage API Server
After=network.target redis.service
Requires=redis.service
```

`After=` controls ordering. Systemd will not start this unit until both `network.target` (network interfaces are up) and `redis.service` (Redis is running) have started. `Requires=` adds a dependency. If `redis.service` fails or stops while `airstage-api.service` is running, systemd will stop `airstage-api.service` too. This enforces the architectural contract: the app cannot run without Redis.

```ini
[Service]
Type=simple
User=deploy
Group=deploy
WorkingDirectory=/opt/airstage
EnvironmentFile=/opt/airstage/.env
ExecStart=/opt/airstage/bin/airstage
```

`Type=simple` means systemd considers the service started as soon as the process starts. The alternative is `Type=notify` where the process signals systemd when it is ready — useful for services with non-trivial startup. Your Go binary starts fast enough that `simple` is correct.

`User=deploy` and `Group=deploy` run the process as the `deploy` user, not root. The `deploy` user has no sudo privileges and owns only `/opt/airstage`. If an attacker compromises the Go binary through a bug, they get a shell as `deploy` — no ability to write to system directories, no access to other users' files, no ability to install system packages. This is the principle of least privilege.

`EnvironmentFile=/opt/airstage/.env` injects every `KEY=VALUE` line from that file as an environment variable for the process. Your Go config loader reads `DATABASE_URL`, `REDIS_URL`, `JWT_SECRET_KEY`, etc. from the environment. The file should be owned by `deploy:deploy` with `chmod 600` (readable only by the owner). Never set secrets as `Environment=` lines directly in the unit file — that file is world-readable by default.

```ini
Restart=always
RestartSec=5s
```

If the process exits for any reason (crash, OOM kill, panic), systemd restarts it after 5 seconds. The 5-second delay prevents a crash loop from slamming external services (Postgres, Redis, LiveKit) with reconnection attempts.

```ini
TimeoutStopSec=25s
KillSignal=SIGTERM
KillMode=process
```

When you run `systemctl stop airstage-api` or `systemctl restart airstage-api`, systemd sends `SIGTERM` to the process. Your `main.go` has `signal.Notify(shutdownCh, syscall.SIGTERM)`, receives the signal, calls `stopWorkers()`, `stopEnforcer()`, then `server.Shutdown(ctx)` with a 20-second timeout. `TimeoutStopSec=25s` gives systemd 25 seconds to wait for the process to exit cleanly before sending `SIGKILL`. The 5-second difference between 20 (app drain) and 25 (systemd timeout) is the safety margin. `KillMode=process` ensures systemd only sends the signal to the main process, not to child processes it may have spawned.

```ini
LimitNOFILE=65536
LimitNPROC=4096
```

`LimitNOFILE` is the maximum number of open file descriptors. On Linux, every network connection is a file descriptor. A Go HTTP server handling 1000 concurrent WebSocket connections uses at least 1000 file descriptors, plus connections to Redis and Postgres. The Linux default limit is 1024. At 1025 connections, your process starts getting "too many open files" errors. Setting this to 65536 is standard practice for any network server. `LimitNPROC` limits the number of OS threads the process can spawn. Go's goroutines are multiplexed over OS threads, but the runtime does create OS threads. 4096 is generous and prevents a runaway goroutine spawn from taking down the machine.

---

## Nginx — Every Directive's Purpose

The configuration has two server blocks. The first handles port 80 (plain HTTP) and does nothing except redirect to HTTPS. This is called an HTTP-to-HTTPS redirect and is mandatory for any public-facing service in 2024+. The `301` is a permanent redirect — browsers and HTTP clients cache it.

In the HTTPS server block:

`ssl_protocols TLSv1.2 TLSv1.3` explicitly disables TLSv1.0 and TLSv1.1, which have known vulnerabilities (BEAST, POODLE). Any client that cannot do TLS 1.2 — which means a browser released before 2013 — cannot connect. That is an acceptable tradeoff for a production API.

`ssl_prefer_server_ciphers on` tells Nginx to use the cipher suite order defined on the server rather than the client's preference. Combined with a curated cipher list, this prevents clients from negotiating weak ciphers even if they claim to prefer them.

The three security headers deserve understanding. `X-Frame-Options: DENY` prevents your site from being embedded in an `<iframe>` on another domain. This blocks a class of attack called clickjacking where an attacker overlays an invisible iframe of your site over a malicious page and tricks users into clicking buttons they cannot see. `X-Content-Type-Options: nosniff` tells browsers not to try to detect the MIME type of a response — use only what the `Content-Type` header says. Without this, a browser might interpret a JSON file as JavaScript if an attacker can control its content. `Strict-Transport-Security: max-age=31536000` tells the browser that for the next year, it must never connect to this domain over plain HTTP, even if a link or redirect points to HTTP. This prevents SSL stripping attacks.

The location blocks:

`location /api/` proxies all HTTP API calls to `127.0.0.1:8080`. The `proxy_http_version 1.1` is necessary for HTTP keepalive — without it, Nginx would use HTTP/1.0 which requires a new TCP connection for every request. `proxy_set_header X-Real-IP $remote_addr` passes the client's real IP to your Go backend, necessary for rate limiting and audit logs. `proxy_read_timeout 30s` means if the backend does not respond within 30 seconds, Nginx closes the connection with a 504. For your API endpoints this is intentionally tight — if your handler takes more than 30 seconds, something is wrong.

`location /api/v1/ws` is the WebSocket upgrade block. WebSocket starts as an HTTP/1.1 request with `Upgrade: websocket` and `Connection: Upgrade` headers. Nginx by default does not forward these headers, which breaks the upgrade. `proxy_set_header Upgrade $http_upgrade` forwards the Upgrade header value from the client. `proxy_set_header Connection "upgrade"` tells Nginx itself to treat this as a connection upgrade. `proxy_read_timeout 3600s` — one hour — keeps the WebSocket connection alive. The default 30s timeout would kill WebSocket connections every 30 seconds if no data is sent.

`location /webhooks/` serves LiveKit webhook calls. These are POST requests from LiveKit Cloud's servers with event payloads. No rate limiting is applied here because you cannot control the rate at which LiveKit sends events, and dropping a webhook means a missed egress_ended event means a missed transcription job.

`location /debug/ { deny all; }` — your Go backend likely exposes `/debug/pprof/` for profiling in development. This block ensures that path is inaccessible from the public internet. Without it, anyone could access your goroutine dump, memory profile, and CPU profile.

`location /` serves static files from `/var/www/airstage` with a fallback to `index.html`. This is the Next.js frontend build output. The `try_files $uri $uri/ /index.html` pattern is the client-side routing fallback — when a user navigates directly to `/rooms/abc123`, Nginx serves `index.html` and the React router handles the client-side route.

---

## The CI/CD Pipeline — How a Git Push Becomes a Running Service

The GitHub Actions workflow triggers on every push to `main`. The sequence is:

`actions/checkout@v4` — clones the repository into the runner's workspace. `actions/setup-go@v5` with `go-version-file: go.mod` reads the Go version from your module file and installs that exact version. The `cache: true` flag caches the Go module download cache between runs, so dependency downloads only happen when go.sum changes — the same cache logic as the Dockerfile.

The build step runs the same flags as `make build-prod`. The CI environment is Ubuntu (amd64), so `GOOS=linux GOARCH=amd64` is technically redundant, but keeping it explicit is correct practice — cross-compilation flags should always be explicit so the command is portable.

`appleboy/scp-action` wraps the `scp` command. It connects to the Droplet using the private SSH key stored in GitHub Secrets, copies `bin/airstage` to `/opt/airstage/bin/` on the server, and strips one path component (`strip_components: 1`) so the file lands at `/opt/airstage/bin/airstage` not `/opt/airstage/bin/bin/airstage`.

`appleboy/ssh-action` runs remote commands over SSH. The restart step does `sudo systemctl restart airstage-api`, waits 3 seconds for the process to settle, then checks `systemctl is-active --quiet airstage-api`. If the service is not active — meaning it crashed on startup — the step fetches the last 30 journal lines and exits with code 1, failing the entire workflow run. This prevents the pipeline from succeeding when the service is actually broken.

The health check step curls the `/api/v1/healthcheck` endpoint. The `-f` flag makes curl exit with a non-zero code if the HTTP status is 4xx or 5xx. This is an end-to-end smoke test — if Nginx is misconfigured, or the Go binary is not listening, or the healthcheck handler is broken, the pipeline fails here.

The three GitHub Secrets you must configure: `DROPLET_IP` is the server's public IP address. `SSH_PRIVATE_KEY` is the full contents of an SSH private key whose public key is in the `deploy` user's `~/.ssh/authorized_keys`. `DOMAIN` is `api.yourdomain.com`, used in the health check curl command.

---

## How to Port This Entire Setup to Any Cloud or VM

The portability of this architecture is the point. Every piece is defined in terms of universal Linux primitives: systemd, Nginx, SSH, a static binary. None of it is DigitalOcean-specific. Here is the mapping:

DigitalOcean Droplet → AWS EC2 instance, Azure VM, GCP Compute Engine VM, Hetzner Cloud server, Vultr instance. They are all Ubuntu (or your preferred distro) Linux machines reachable via SSH. The `setup.sh` script runs identically on any of them.

DigitalOcean Spaces → AWS S3, Azure Blob Storage, GCP Cloud Storage. The `SpacesConfig` in your Go code uses an S3-compatible endpoint. You change `SPACES_ENDPOINT`, `SPACES_ACCESS_KEY`, and `SPACES_SECRET_KEY`. No code changes.

Neon Postgres → AWS RDS, Azure Database for PostgreSQL, GCP Cloud SQL, Supabase. They all expose a `DATABASE_URL` connection string. You change one environment variable.

LiveKit Cloud → Self-hosted LiveKit (on a separate VM), AWS-hosted LiveKit. The `LIVEKIT_HOST`, `LIVEKIT_API_KEY`, and `LIVEKIT_API_SECRET` change. No code changes.

GitHub Actions → GitLab CI, Bitbucket Pipelines, CircleCI. The primitives are the same: checkout, build binary, scp, ssh. You rewrite the YAML syntax but not the logic.

The binary is cross-compiled, so the CI runner's architecture does not matter. You can run GitHub Actions on an M3 Mac runner and produce a binary for a Graviton ARM instance by changing `GOARCH=amd64` to `GOARCH=arm64`.

---

## Understanding Docker Networking — the Model That Transfers Everywhere

When Docker Compose starts services, it creates a virtual network. By default, every service in a `docker-compose.yml` is attached to a network named `<project>_default`. Services on this network can reach each other by service name as the hostname. If you had both the Go app and Redis in Compose, your app would connect to Redis at `redis:6379` — the service name resolves to the container's IP on the virtual network.

This virtual network is implemented using Linux network namespaces and a virtual Ethernet bridge. Docker creates a bridge interface (usually `docker0` or `br-<id>`) on the host. Each container gets a virtual ethernet pair (veth pair) — one end inside the container's network namespace, one end attached to the bridge on the host. Packets between containers flow through the bridge without leaving the host. Packets to the outside world go through the bridge, through iptables NAT rules Docker maintains, and out the host's real network interface.

This understanding transfers to cloud networking. AWS VPC, Azure VNet, GCP VPC — these are all the same concept at the hypervisor level: a virtual L2 network segment where machines communicate by private IP. Security groups and network security groups are iptables rules managed by the cloud control plane rather than Docker. The mental model is identical.

Port binding — `"127.0.0.1:6379:6379"` — creates an iptables DNAT rule that redirects packets arriving on host `127.0.0.1:6379` into the container's network namespace at `6379`. Publishing a port on `0.0.0.0` instead of `127.0.0.1` creates the DNAT rule for all host interfaces, making the service publicly accessible on the machine's public IP. This is why `"127.0.0.1:6379:6379"` is the security-correct form for development — it replicates the production Redis binding.

---

## Future Scope — What to Do When Specific Things Happen

**When traffic grows beyond what one Droplet handles:** The first move is not more servers. It is profiling. `go tool pprof` against the running binary gives you a CPU and memory profile. Most Go services have one hot path. Fix that first. If after profiling the single machine is genuinely saturated, you add a load balancer in front of two or more Droplets. DigitalOcean Load Balancer, AWS ALB, Nginx on a separate machine, HAProxy — they all terminate HTTP and distribute requests round-robin or by least-connections. The Go binary is stateless (session state is in Redis, data in Postgres) so any request can go to any instance. You do not need sticky sessions.

**When you need WebSocket stickiness across multiple instances:** A WebSocket connection is stateful — it is a persistent TCP connection to one specific server. If you add a second server, you cannot load-balance a WebSocket mid-session. The architecture already handles this correctly: the `realtime.Hub` publishes messages via Redis PubSub. When a client sends a message, it goes to the server it is connected to, which publishes to Redis, which is received by all servers, and each server pushes to its locally-connected clients. No server needs to know which clients are on which server. Adding a second server requires only: SSH in, copy binary, run it on port 8081, add it to Nginx upstream block.

**When you need per-environment (staging/production) isolation:** You add a staging Droplet. The GitHub Actions workflow gets a second job or a separate workflow triggered by pushes to a `staging` branch. The secrets for staging (`STAGING_DROPLET_IP`, `STAGING_SSH_PRIVATE_KEY`, `STAGING_DOMAIN`) are separate from production. The `setup.sh` script runs identically on the staging machine.

**When the binary has environment-specific initialization (e.g., database migrations):** Add a migration step to the CI pipeline between the deploy step and the reload step. The SSH action runs `cd /opt/airstage && ./bin/airstage --migrate-up` (you would add a `--migrate-up` flag to `main.go`) before restarting the service. Migrations run in the CI runner's SSH session, fail fast if the schema change breaks something, and the pipeline fails before the new binary goes live.

**When you need to run the full stack in Docker for integration testing in CI:** Add a `docker-compose.test.yml` that runs both the Go app container and Redis in Compose. The workflow adds a step `docker compose -f docker-compose.test.yml up -d && go test ./... && docker compose -f docker-compose.test.yml down`. The Go app container uses the same `Dockerfile` built in the same pipeline step, ensuring the binary under test is identical to the binary being deployed.

**When `FROM scratch` causes problems (e.g., you need timezone data or a `/tmp` directory):** Switch the final stage to `FROM gcr.io/distroless/static:nonroot`. Distroless is Google's minimal image that includes ca-certificates, timezone data, and a non-root user but no shell. It is slightly larger than scratch (~2MB) but handles these edge cases. The `nonroot` tag runs as UID 65532 instead of root, equivalent to the `deploy` user principle in systemd.

**When you want image storage and versioning across cloud environments:** Push to a container registry. Docker Hub is public. GitHub Container Registry (`ghcr.io`) is free for public repositories and cheap for private. AWS ECR, Azure ACR, GCP Artifact Registry are cloud-native options. The push command is `docker push recallo/api:abc1234` after a `docker login`. The deploy process then becomes `docker pull` on the target server and `docker run` instead of `scp`. This is the bridge to container orchestration if you ever need it.

**When you need the app container to actually run in production via Docker:** The pattern is: Docker pulls the image on the Droplet, runs it with `docker run -d --restart=unless-stopped --env-file /opt/airstage/.env -p 127.0.0.1:8080:8080 recallo/api:abc1234`. Nginx still proxies to `127.0.0.1:8080`. Redis still runs as a native service. The systemd unit is replaced by Docker's own restart policy. The GitHub Actions deploy step changes from `systemctl restart` to `docker pull && docker stop && docker run`. The operational model is nearly identical.

---

## The One Thing That Takes Everything Else for Granted

Every part of this system — the scratch image, the static binary, the systemd unit, the Nginx config, the CI pipeline — is built around one principle: **the artifact of a build is immutable and the environment it needs is declared, not assumed.**

The Dockerfile declares exactly what goes into the image. The systemd unit declares what environment variables the binary gets. The Nginx config declares what traffic flows where. The CI pipeline declares the exact steps to go from source to running service. None of it depends on the state of the machine at deployment time, the tools installed by hand by a previous engineer, or conventions undocumented anywhere.

When you understand that principle at the cellular level, you can evaluate any new deployment technology — Docker Swarm, Nomad, any future thing — as an answer to the same question: does this tool let me declare my artifact and its environment more clearly and reproducibly than what I have now? If yes, it is worth learning. If no, you already have a better solution.
