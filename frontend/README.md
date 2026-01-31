# GTO Game Client

Vue 3 frontend for the multiplayer drawing and guessing game.

## Features

- **Real-time canvas drawing** with custom Dende engine
- **WebSocket communication** using Protocol Buffers
- **Drawing tools**: freehand, bucket fill, undo/redo
- **Live chat** with guess detection
- **Turn-based gameplay** with word selection
- **Score tracking** and leaderboards
- **Shareable game rooms** via invite links

## Tech Stack

- Vue 3 + TypeScript
- Vite
- Tailwind CSS 4
- Protocol Buffers
- Custom canvas library (Dende)

## Getting Started

### Prerequisites

- Node.js 18+
- pnpm (recommended) or npm

### Installation

```bash
# Install dependencies
pnpm install

# Start development server
pnpm dev

# App runs on http://localhost:3000
```

### Build for Production

```bash
pnpm build
```

## Project Structure

```
src/
├── views/              # Page components
│   ├── HomeView.vue
│   ├── LoginView.vue
│   ├── SignUpView.vue
│   └── GameView.vue   # Main game interface
├── proto/             # Protocol Buffer definitions
│   ├── serialization.proto
│   └── drawing.proto
├── dende.ts           # Custom canvas drawing engine
├── router/            # Vue Router configuration
└── config.ts          # API endpoint configuration
```

## Dende Canvas Engine

Custom drawing library with:
- **Dual modes**: Drawing and bucket fill
- **Undo/Redo stacks** with configurable history depth
- **Batched updates** for efficient network transmission
- **High-DPI support** for Retina displays
- **Event-driven API** for network synchronization

Example usage:
```typescript
const dende = new Dende(800, 600)
dende.enableDrawing()
dende.setLineColorRGBA(255, 0, 0, 1)
dende.onPartCreated((part) => {
  sendToServer(part)
})
```

## Configuration

Update `src/config.ts` with your backend URL:

```typescript
export const API_BASE_URL = 'https://api.yourdomain.com';
```

## Game Flow

1. **Login/Signup** → Receive JWT cookie
2. **Matchmaking** → Connect via WebSocket
3. **Word Selection** → Drawer picks from 3 words
4. **Drawing Round** → 80 seconds to draw
5. **Guessing** → Chat messages checked against word
6. **Turn Summary** → Score updates after each round
7. **Leaderboard** → Final scores after 3 rounds

## License

MIT