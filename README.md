# GTO Game Server

Real-time multiplayer drawing and guessing game backend built with Go and WebSockets.

## Features

- **WebSocket-based real-time communication** using Protocol Buffers
- **Heap-based matchmaking system** for efficient room assignment
- **Turn-based game state management** with multiple rounds
- **Rate limiting** to prevent message spam
- **JWT authentication** with httpOnly cookies
- **PostgreSQL database** for user management

## Tech Stack

- Go 1.25
- Gin web framework
- Gorilla WebSocket
- Protocol Buffers
- PostgreSQL
- Docker & Docker Compose

## Getting Started

### Prerequisites

- Docker and Docker Compose
- Go 1.25+ (for local development)

### Development

```bash
# Copy environment variables
cp .env.example .env

# Start development server
docker compose up

# Server runs on http://localhost:5000
```

### Production

```bash
# Build and run production containers
docker compose -f docker-compose.production.yml up -d
```

## API Endpoints

### Authentication
- `POST /authentication/signup` - Create new account
- `POST /authentication/login` - Login with credentials
- `POST /authentication/logout` - Logout

### Game
- `GET /matchmaking` - WebSocket endpoint for automatic matchmaking
- `GET /join/:gameid` - WebSocket endpoint to join specific game room

## Project Structure

```
api/
├── cmd/server/          # Application entry point
├── internal/
│   ├── authentication/  # Auth handlers and JWT logic
│   ├── game/           # Game logic, matchmaking, WebSocket handlers
│   │   ├── gameroom.go # Game state management
│   │   ├── matchmaking.go # Heap-based room assignment
│   │   ├── player.go   # Player connection handling
│   │   └── *.proto     # Protocol Buffer definitions
│   └── shared/
│       ├── authorization/ # JWT verification
│       ├── configs/    # Configuration constants
│       ├── database/   # PostgreSQL queries
│       └── logger/     # Logging utilities
└── words.txt           # Word list for drawing
```

## Game Protocol

Messages are sent as binary data with Protocol Buffers, each with a type marker:

- `SERIAL_EVENT` - Game events (player joined, game started, etc.)
- `SERIAL_MESSAGE` - Chat messages
- `SERIAL_DRAWING` - Drawing data with coordinates and style
- `SERIAL_WORD_CHOICE` - Drawer's word selection
- `SERIAL_TURN_SUMMARY` - Round results and scores

## Configuration

Environment variables in `.env`:

```bash
POSTGRES_USER=myuser
POSTGRES_PASSWORD=mypassword
POSTGRES_DB=drawing_app
JWT_KEY=your-secret-key-here
FRONTEND_ORIGIN=localhost:3000
```

## License

MIT