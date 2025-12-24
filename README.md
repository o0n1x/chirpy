

# Chirpy API

A Twitter-like REST API built with Go as part of the [Boot.dev](https://boot.dev) backend learning path. Chirpy allows users to create accounts, post short messages (chirps), and manage authentication with JWT tokens.

## Features

- User registration and authentication with JWT tokens
- Refresh token support for extended sessions
- Create, read, and delete chirps (140 character limit)
- Profanity filter for chirp content
- Chirpy Red premium membership via Polka webhook integration
- Admin metrics and reset functionality
- Static file serving

## Tech Stack

- **Language:** Go
- **Database:** PostgreSQL
- **Password Hashing:** Argon2id (via [alexedwards/argon2id](https://github.com/alexedwards/argon2id))
- **Authentication:** JWT (JSON Web Tokens) with refresh tokens
- **Database Queries:** sqlc
- **Environment Management:** godotenv

## Prerequisites

- Go 1.21+
- PostgreSQL
- [sqlc](https://sqlc.dev/) (for generating database code)
- [goose](https://github.com/pressly/goose) (for database migrations)

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/o0n1x/chirpy.git
   cd chirpy
    ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Set up your environment variables by creating a `.env` file:
   ```env
   DB_URL=postgres://username:password@localhost:5432/chirpy?sslmode=disable
   SECRET_JWT=your-secret-key-here
   POLKA_KEY=your-polka-api-key
   PLATFORM=dev
   ```

4. Run database migrations:
   ```bash
   goose -dir sql/schema postgres "your-db-url" up
   ```


5. Run the server:
   ```bash
   go run .
   ```

The server will start on port `8080`.

## API Endpoints

### Health Check

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/healthz` | Health check endpoint |

### Users

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| POST | `/api/users` | Create a new user | No |
| PUT | `/api/users` | Update user email/password | JWT |
| POST | `/api/login` | Login and receive tokens | Password |
| POST | `/api/refresh` | Refresh access token | Refresh Token |
| POST | `/api/revoke` | Revoke refresh token | No |

### Chirps

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| POST | `/api/chirps` | Create a new chirp | JWT |
| GET | `/api/chirps` | Get all chirps | No |
| GET | `/api/chirps/{id}` | Get a specific chirp | No |
| DELETE | `/api/chirps/{id}` | Delete a chirp | JWT |

**Query Parameters for GET /api/chirps:**
- `author_id` - Filter chirps by author UUID
- `sort` - Sort order (`asc` or `desc` by creation date)

### Webhooks

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| POST | `/api/polka/webhooks` | Polka payment webhook | API Key |

### Admin

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/admin/metrics` | View file server hit count |
| POST | `/admin/reset` | Reset metrics and users (dev only) |

## Request/Response Examples

### Create User
```bash
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "secret123"}'
```

### Login
```bash
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "secret123"}'
```

### Create Chirp
```bash
curl -X POST http://localhost:8080/api/chirps \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-jwt-token>" \
  -d '{"body": "Hello, Chirpy!"}'
```

## Project Structure

```
chirpy/
├── main.go
├── index.html
├── internal/
│   ├── api/
│   │   └── api.go
│   ├── auth/
│   │   └── auth.go
│   │   └── jwt.go
│   │   └── refresh_token.go
│   └── database/
│       └── (sqlc generated files)
├── sql/
│   ├── schema/
│   │   └── (migration files)
│   └── queries/
│       └── (query files)
├── .env
├── go.mod
├── go.sum
└── README.md
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `DB_URL` | PostgreSQL connection string |
| `SECRET_JWT` | Secret key for JWT signing |
| `POLKA_KEY` | API key for Polka webhook authentication |
| `PLATFORM` | Set to `dev` to enable admin reset functionality |

## Content Moderation

Chirpy automatically filters the following words, replacing them with `****`:
- kerfuffle
- sharbert
- fornax

## License

This project was built as part of the Boot.dev curriculum.


