# Skill: Docker Containers

## Overview

Docker is a platform for building, shipping, and running applications in lightweight, portable containers. This skill covers image creation, container lifecycle management, networking, and multi-container applications.

## Key Concepts

- **Image**: A read-only template built from a `Dockerfile`; the blueprint for containers.
- **Container**: A runnable instance of an image.
- **Dockerfile**: A text file with instructions for building an image.
- **Registry**: A storage service for images (e.g., Docker Hub, GitHub Container Registry).
- **Volume**: A persistent storage mechanism that lives outside the container filesystem.
- **Network**: A virtual network connecting containers.
- **Compose**: A tool (`docker compose`) for defining and running multi-container applications.

## Common Tasks

### Build an image
```bash
docker build -t my-app:latest .
docker build --no-cache -t my-app:latest .   # skip layer cache
```

### Run a container
```bash
docker run -d --name my-app -p 8080:80 my-app:latest
docker run --rm -it ubuntu:22.04 bash         # interactive, auto-remove
```

### List and manage containers
```bash
docker ps                     # running containers
docker ps -a                  # all containers
docker stop my-app
docker rm my-app
docker logs -f my-app         # follow logs
docker exec -it my-app bash   # open shell in running container
```

### Manage images
```bash
docker images
docker pull nginx:alpine
docker rmi my-app:latest
docker image prune            # remove dangling images
```

### Push an image to a registry
```bash
docker tag my-app:latest ghcr.io/<owner>/my-app:latest
docker push ghcr.io/<owner>/my-app:latest
```

### Use Docker Compose
```bash
docker compose up -d          # start all services in background
docker compose down           # stop and remove containers
docker compose logs -f web    # follow logs for the "web" service
docker compose build          # rebuild images
```

## Example Dockerfile (multi-stage, Python)

```dockerfile
# --- build stage ---
FROM python:3.12-slim AS builder
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir --prefix=/install -r requirements.txt

# --- runtime stage ---
FROM python:3.12-slim
WORKDIR /app
COPY --from=builder /install /usr/local
COPY . .
USER nobody
CMD ["python", "main.py"]
```

## Best Practices

- Use specific image tags (not `:latest`) in production for reproducibility.
- Run containers as a non-root user.
- Use multi-stage builds to keep final images small.
- Never store secrets in the image; use environment variables or secrets managers.
- Set `HEALTHCHECK` instructions so orchestrators can detect unhealthy containers.
- Pin base image digests for critical workloads (`FROM ubuntu@sha256:...`).
- Use `.dockerignore` to exclude unnecessary files from the build context.

## References

- [Docker Documentation](https://docs.docker.com/)
- [Dockerfile Best Practices](https://docs.docker.com/develop/develop-images/dockerfile_best-practices/)
- [Docker Compose Reference](https://docs.docker.com/compose/compose-file/)
