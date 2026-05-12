# DriveHive

**DriveHive** is a Discord-like desktop community app for friends and small groups. Create communities, chat in channels, manage your space, and connect in real time.

- **Backend:** Go (Golang) — fast concurrency and efficient RAM usage  
- **Desktop App:** Tauri — smaller app size and native OS webview  
- **Frontend:** Svelte — compile-time UI for snappy performance  
- **Database:** SQLite — low-overhead setup for small groups  

---

## Table of Contents
- [Prerequisites](#prerequisites)
- [Project Structure](#project-structure)
- [Architecture Overview](#architecture-overview)
- [Getting Started](#getting-started)
  - [1) Run the backend](#1-run-the-backend)
  - [2) Run the frontend (Tauri)](#2-run-the-frontend-tauri)
- [Configuration](#configuration)
- [Database](#database)
- [Development Scripts](#development-scripts)
- [Roadmap](#roadmap)
- [Contributing](#contributing)
- [License](#license)
- [Contact](#contact)

---

## Prerequisites
- Go (latest stable recommended)
- Node.js + npm
- Rust
- Tauri CLI / tooling

---

## Project Structure
drive-hive/
├── backend/                # Go source code
│   ├── cmd/                # Entry points for the application
│   │   └── server/         # main.go lives here
│   ├── internal/           # Private application code (not importable by others)
│   │   ├── api/            # HTTP/WebSocket handlers
│   │   ├── auth/           # JWT, hashing, and session logic
│   │   ├── database/       # SQLite connection and migrations
│   │   └── models/         # Structs for Users, Messages, and Channels
│   ├── pkg/                # Publicly importable utility packages
│   ├── go.mod
│   └── go.sum
├── frontend/               # Tauri + Svelte application
│   ├── src/                # Svelte frontend source
│   │   ├── lib/            # Reusable Svelte components (ChatBox, Sidebar)
│   │   ├── routes/         # UI Views/Pages
│   │   └── stores/         # Client-side state management (using Svelte stores)
│   ├── src-tauri/          # Rust backend for the Desktop app
│   │   ├── icons/          # App icons
│   │   ├── src/            # Rust logic (system tray, window management)
│   │   ├── tauri.conf.json # Tauri configuration
│   │   └── Cargo.toml      # Rust dependencies
│   ├── public/             # Static assets (images, fonts)
│   ├── package.json
│   └── svelte.config.js
├── shared/                 # Shared resources (Optional)
│   └── proto/              # If using Protocol Buffers for communication
├── scripts/                # Build and deployment scripts
├── docker-compose.yml      # For running the database or server in a container
└── README.md

---

## Architecture Overview
Backend (Go)

    Serves HTTP/WebSocket endpoints for real-time chat
    Handles authentication using JWT
    Uses SQLite for persistence
    Organized into:
        internal/api/ (HTTP/WebSocket handlers)
        internal/auth/ (JWT, hashing, session logic)
        internal/database/ (SQLite connection + migrations)
        internal/models/ (data structs for users/messages/channels)

Frontend (Tauri + Svelte)

    Desktop UI packaged with Tauri
    UI built with Svelte
    App state via Svelte stores
    Rust side handles native OS features (system tray, window management)

Database (SQLite)

    Low-overhead and ideal for smaller friend groups.

---

## Getting Started

## 1) Run the backend

From the backend/ directory:
bash

cd backend
go run ./cmd/server

If your backend requires migrations, run them according to how your migration code is set up in internal/database/.
## 2) Run the frontend (Tauri)

From the frontend/ directory:
bash

cd frontend
npm install
npm run dev

Launch the app using the Tauri output instructions.

## Configuration

Set any needed environment variables for the backend and/or frontend (ports, JWT secret, database path, etc.).

If you have a file like .env.example, add it here and mention how to use it.

## Database

SQLite is configured through the backend database layer.

Typical flow:

    Apply migrations (if applicable)
    Start the backend
    Chat/community data persists automatically in SQLite

---

## Development Scripts

None currently 

---

## Roadmap

    Authentication + user accounts
    Communities/servers
    Channels
    Real-time messaging
    Notifications/mentions
    Roles + permissions
    Moderation tools
    Media/file sharing

---

## Contributing

    Fork the repository
    Create a feature branch:
    bash

    git checkout -b feature/your-feature

    Commit changes:
    bash

    git commit -m "Add your feature"

    Push:
    bash

    git push origin feature/your-feature

    Open a Pull Request

---

## License

Copyright © 2026 <Hardrive Technologies>

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the “Software”), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

---

## Contact

[Hardrive Technologies](https://github.com/Hardrive-Technologies)
[Hardrivetech](https://github.com/Hardrivetech)

