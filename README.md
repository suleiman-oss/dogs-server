# 🐕 Dogs Breed Registry — Go REST API + GUI

A full-stack CRUD application for managing dog breeds and sub-breeds. Zero external dependencies — built entirely with Go's standard library and a hand-rolled frontend.

---

## Stack

| Layer    | Tech                              |
|----------|-----------------------------------|
| Language | Go (see `go.mod` for version)     |
| Server   | `net/http` (stdlib only, no Gin)  |
| Storage  | JSON file on disk (thread-safe)   |
| Frontend | Vanilla HTML / CSS / JS           |
| Deploy   | Docker → Render or Fly.io         |

---

## Running Locally (without Docker)

**Requirements:** Go installed (matching the version in `go.mod`)

```bash
# 1. Clone
git clone https://github.com/suleiman-oss/dogs-server.git
cd dogs-server

# 2. Run directly
go run ./cmd/server
# → http://localhost:3000

# Or build a binary
go build -o bin/dogs-server ./cmd/server
./bin/dogs-server
# → http://localhost:3000
```

On first run, the server will:
- ensure `data/dogs.json` exists
- seed it from `data/seed.json` if it doesn't

---

## Running with Docker

### Build the image

From the project root (where the `Dockerfile` lives):

```bash
docker build -t dogs-api .
```

This multi-stage build:
- compiles the Go binary
- copies `data/seed.json` and `frontend/public/` into the runtime image
- exposes port `3000`

### Run the container

**Simple run** (data stored inside the container — lost on removal):

```bash
docker run --rm -p 3000:3000 dogs-api
# → http://localhost:3000
```

**With persistent data** (recommended — changes survive restarts):

```bash
mkdir -p ./data
docker run -p 3000:3000 -v "$(pwd)/data:/app/data" dogs-api

# Windows PowerShell:
docker run -p 3000:3000 -v ${PWD}/data:/app/data dogs-api

# Windows CMD:
docker run -p 3000:3000 -v %cd%/data:/app/data dogs-api
```

On first run with an empty `./data` directory, the app will copy `seed.json` → `dogs.json` and use that file for all subsequent reads and writes.

### Using a published image

```bash
docker pull suleimanoss/dogs-server:latest
docker run -p 3000:3000 youruser/dogs-server:latest
# → http://localhost:3000

# With persistent data:
mkdir -p ./dogs-data
docker run -p 3000:3000 -v "$(pwd)/dogs-data:/app/data" youruser/dogs-server:latest
```

---

## Project Structure

```
dogs-server/
├── cmd/
│   └── server/
│       └── main.go              ← entrypoint, wires everything together
├── internal/
│   ├── handler/
│   │   └── handler.go           ← all HTTP route handlers
│   └── store/
│       └── store.go             ← thread-safe JSON store
├── frontend/
│   └── public/
│       └── index.html           ← full GUI (served by Go)
├── data/
│   └── seed.json                ← initial dog data (copied to dogs.json on first run)
├── Dockerfile                   ← multi-stage build
├── render.yaml                  ← Render.com deploy config
├── fly.toml                     ← Fly.io deploy config
└── go.mod
```

---

## REST API Reference

Base URL: `http://localhost:3000/api/dogs`

### GET /api/dogs

Returns all breeds and sub-breeds.

```bash
curl http://localhost:3000/api/dogs
```

```json
{
  "status": "success",
  "data": {
    "labrador": [],
    "poodle": ["miniature", "standard", "toy"]
  }
}
```

---

### GET /api/dogs/:breed

Returns a single breed and its sub-breeds.

```bash
curl http://localhost:3000/api/dogs/poodle
```

```json
{
  "status": "success",
  "breed": "poodle",
  "subBreeds": ["miniature", "standard", "toy"]
}
```

---

### POST /api/dogs — create a breed

```bash
curl -X POST http://localhost:3000/api/dogs \
  -H "Content-Type: application/json" \
  -d '{"breed": "goldendoodle", "subBreeds": ["miniature", "standard"]}'
```

---

### PUT /api/dogs/:breed — replace sub-breeds

Fully overwrites the sub-breeds list for an existing breed.

```bash
curl -X PUT http://localhost:3000/api/dogs/poodle \
  -H "Content-Type: application/json" \
  -d '{"subBreeds": ["teacup", "miniature", "standard"]}'
```

---

### PATCH /api/dogs/:breed — add sub-breeds

Appends sub-breeds without removing existing ones.

```bash
curl -X PATCH http://localhost:3000/api/dogs/poodle \
  -H "Content-Type: application/json" \
  -d '{"subBreeds": ["giant"]}'
```

---

### DELETE /api/dogs/:breed — delete a breed

```bash
curl -X DELETE http://localhost:3000/api/dogs/pug
```

---

### DELETE /api/dogs/:breed/:subbreed — delete a sub-breed

```bash
curl -X DELETE http://localhost:3000/api/dogs/poodle/toy
```