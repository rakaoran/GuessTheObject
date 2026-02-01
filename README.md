# GuessTheObject

A real-time multiplayer drawing and guessing game inspired by Skribbl.io. Players join rooms to take turns drawing a word while others guess it in real-time.

## Overview

GuessTheObject connects multiple players in a live game session via WebSockets. It features public and private lobbies, live chat, and a custom drawing engine optimized for performance and smooth synchronization.

The project is split into a **Vue 3** frontend and a **Go** backend, deployed using Docker and Nginx.

## Key Features

- **Real-time Gameplay**: Instant synchronization of drawing strokes and chat messages using WebSockets.
- **Custom Drawing Engine**: Powered by the npm package [`@rakaoran/dende`](https://www.npmjs.com/package/@rakaoran/dende), a lightweight canvas engine I built and published to npm (lol) specifically for this project.
- **Lobby System**: Support for creating private and public rooms, and joining rooms with a code or from a list of public rooms.
- **Fair Play**: Authoritative server architecture that validates every action (drawing, guessing) to ensure no cheating.
- **Authentication**: Secure signup and login flow using JWTs and Argon2id hashing.


## Architecture

### Frontend
- **Framework**: Vue 3 + Vite + TypeScript
- **Styling**: TailwindCSS 4
- **State/Network**: Protobufs for structured data exchange.
- **Drawing**: `@rakaoran/dende` handling canvas interactions and stroke serialization.

### Backend
- **Language**: Go 1.25
- **Web Framework**: Gin
- **Concurrency**: Implements the **Actor Model** to manage game state safely without mutex locks.
- **Communication**: Gorilla WebSocket for real-time events.
- **Database**: PostgreSQL (managed via `pgx`).
- **Testing**: Robust suite including **integration tests** with **Testcontainers** and extensive use of **table-driven tests**.

### Infrastructure
- **Containerization**: Docker & Docker Swarm compatibility (`stack.yml`).
- **Proxy**: Nginx as a reverse proxy, SSL terminator, and **rate limiter** (1r/s) to prevent abuse.
- **SSL**: Automated Let's Encrypt certificates via Certbot.

## Development Setup

### 1. Configuration
Navigate to the `backend/` directory and create your `.env` file based on the example:
```bash
cd backend
cp .env.example .env
```

### 2. Start Services
Run the backend and databases using Docker Compose:
```bash
docker compose up
```

The backend will be available at `http://localhost:5000`.

### 3. Frontend
Navigate to the `frontend/` directory, install dependencies, and start the dev server:
```bash
cd frontend
pnpm install
pnpm dev
```
The frontend will be available at `http://localhost:3000`.

## Production Deployment

### 1. Configuration
Create a `.env` file with production values like in `.env.example` and export them to your shell environment:
```bash
export $(cat .env | xargs)
```

### 2. SSL & Deployment
Run the following sequence to generate certificates and deploy the stack:

1. **Generate Dummy Certs** (first time only):
   ```bash
   make dummy-cert
   ```

2. **Deploy Stack**:
   ```bash
   docker stack deploy -c stack.yml gto
   ```

3. **Start Certbot Loop** (for auto-renewal):
Wait for the stack to be deployed, then run:
   ```bash
   make run-certbot EMAIL=your@email.com
   ```

## CI/CD Local Testing

To test the GitHub Actions workflows locally, use [act](https://github.com/nektos/act).

1. **Configure Secrets**:
   Create a `.secrets` file based on the example:
   ```bash
   cp .secrets.example .secrets
   ```
   Fill in your Docker Hub credentials and VPS details.

2. **Run Workflows**:
   ```bash
   act --secret-file .secrets
   ```

### 3. Frontend Build
For the frontend, build the static assets:
```bash
cd frontend
pnpm build
```
The deployment assumes these assets are served by Nginx or a similar static host (e.g., Cloudflare Pages).
