# Playlist Go

Discover music videos, collect your favorite tracks, and organize playlists through a responsive web interface powered by Go and React.

Playlist Go uses the official YouTube Data API to retrieve track metadata and MongoDB to persist playlists and cached searches. It is a work in progress built around music discovery and playlist organization.

## What you can do

- Search YouTube music videos and view titles, channels, thumbnails, and durations.
- Select tracks into a temporary queue before saving them.
- Create playlists or add the selection to an existing playlist.
- Remove tracks and change the order of saved playlist tracks.
- Browse playlists by name, creation date, or latest update.
- Use the responsive interface on desktop, tablet, or mobile.
- Reuse cached search results for up to 72 hours and track the application's daily budget for new searches.

Playlists are stored in this application's database. The current version does not play audio or export playlists to a YouTube account. The interface is currently in Brazilian Portuguese.

## Built with

| Layer | Technologies |
| --- | --- |
| API | Go, standard library HTTP server |
| Database | MongoDB 7, MongoDB Go Driver v2 |
| YouTube integration | YouTube Data API v3, Google OAuth 2.0 |
| Web interface | React 19, TypeScript, Vite 6 |
| Styling and interactions | CSS Modules, Framer Motion |
| Local infrastructure | Docker Compose |
| Backend packaging | Multi-stage Docker build |

## Requirements

Before starting, install:

- [Go](https://go.dev/dl/) **1.25 or newer**.
- [Node.js](https://nodejs.org/) **22 or newer**, with npm.
- [Docker](https://docs.docker.com/get-started/get-docker/) with Docker Compose, or access to an existing MongoDB instance.
- Git and a web browser.

You also need a Google account, a Google Cloud project with **YouTube Data API v3 enabled**, and an **OAuth client of type Desktop app**. Each person running the project must supply their own credentials; none are included in this repository.

## Run locally

### 1. Get the project

```bash
git clone https://github.com/VirgilioFuchs/PlaylistGo.git
cd PlaylistGo
```

### 2. Configure Google OAuth

1. Create or select a project in the [Google Cloud Console](https://console.cloud.google.com/).
2. Enable **YouTube Data API v3** in the API Library.
3. Configure the OAuth consent screen. If the application is in testing, add the Google account you will use as a test user.
4. Create an OAuth client with the application type **Desktop app**.
5. Copy the client ID and client secret into your local `.env` file in the next step.

The backend uses a loopback callback on a dynamically assigned port at `127.0.0.1`. A Desktop OAuth client fits this flow; a Web application client with an unrelated redirect URI will not.

See Google's [OAuth guide for desktop applications](https://developers.google.com/youtube/v3/guides/auth/installed-apps) for details. The current implementation requests the `https://www.googleapis.com/auth/youtube` scope.

### 3. Create your local configuration

From the repository root:

```bash
cp .env.example .env
```

On Windows PowerShell:

```powershell
Copy-Item .env.example .env
```

Edit `.env` and replace the `CLIENT_ID` and `CLIENT_SECRET` placeholders with your own values. Keep the default MongoDB URI for the included Compose service.

The Go API loads the root `.env` automatically. Values already set in the process environment take precedence.

### 4. Start MongoDB and the API

```bash
docker compose up -d mongodb
go mod download
go run ./cmd/api
```

On the first run, the backend opens a browser for Google authorization, or prints an authorization URL in the terminal. Complete this step **on the same computer that is running the API**. The server waits for authorization before it becomes available.

After successful authorization, the application creates `youtube_token.json` locally and listens at **http://localhost:8080**. Keep that token private.

Check the API:

```bash
curl http://localhost:8080/healthz
```

### 5. Start the web interface

Open another terminal:

```bash
cd PlaylistGo/web
npm ci
npm run dev
```

If you are already in the repository root, use `cd web` instead.

Open **http://localhost:5173**. In development, Vite proxies `/api` requests to `http://127.0.0.1:8080`, so no frontend environment file is needed for the default setup.

## Configuration

| Variable | Purpose | Example or default |
| --- | --- | --- |
| `APP_ENV` | Environment name reported by the API | `development` |
| `PORT` | API HTTP port | `8080` |
| `MONGODB_URI` | MongoDB connection URI, including the database name | `mongodb://127.0.0.1:27017/musicgo` |
| `CLIENT_ID` | Your Google Desktop OAuth client ID | Required |
| `CLIENT_SECRET` | Your Google OAuth client secret | Required |
| `YOUTUBE_TOKEN_PATH` | Local path for the OAuth token file | `youtube_token.json` |
| `YOUTUBE_SEARCH_DAILY_BUDGET` | Application budget for uncached external searches | `100` |
| `FRONTEND_ORIGIN` | Frontend origin configuration value | `http://localhost:5173` |
| `VITE_API_BASE_URL` | Optional frontend API URL, read by Vite at build time | Empty for the development proxy |

For the frontend, set `VITE_API_BASE_URL` in `web/.env.local` or the frontend build environment if you need it. The API's root `.env` does not automatically configure Vite.

`FRONTEND_ORIGIN` is loaded by the backend, but the current HTTP router does not implement CORS middleware. Use a same-origin proxy for the frontend and API, or configure CORS at your reverse proxy before using a separate frontend origin.

The daily budget is an application counter for uncached searches, shared by the API instance's database. It is **not** a measurement of the Google project's exact quota usage. Valid cached queries remain available when this application budget is exhausted; expired entries need a new external search.

## API overview

| Method | Endpoint | Action |
| --- | --- | --- |
| `GET` | `/healthz` | Check API health |
| `GET` | `/api/search?q=...` | Search tracks |
| `GET` | `/api/playlists?sort=recent` | List playlists; also supports `name` and `updated` |
| `POST` | `/api/playlists` | Create a playlist |
| `POST` | `/api/playlists/{playlistId}/tracks` | Add a track snapshot |
| `DELETE` | `/api/playlists/{playlistId}/tracks/{videoId}` | Remove a track |
| `PUT` | `/api/playlists/{playlistId}/tracks/order` | Reorder tracks with an expected revision |

Search example:

```bash
curl "http://localhost:8080/api/search?q=the%20cure"
```

Create a playlist:

```bash
curl -X POST http://localhost:8080/api/playlists \
  -H "Content-Type: application/json" \
  -d '{"name":"Road Trip"}'
```

Use a complete track object returned by search when adding a track. Reordering accepts `trackIds` and `expectedRevision`; the revision check helps detect concurrent changes.

Errors use structured JSON codes such as `validation_error`, `conflict`, `not_found`, `quota_limited`, and `upstream_unavailable`. Empty search queries return HTTP 400; exhausted new-search budgets return HTTP 429.

## Build and check

Backend:

```bash
go test ./...
go vet ./...
go build ./cmd/api
```

Frontend:

```bash
cd web
npm ci
npm run build
```

The frontend build runs TypeScript checks and writes the production files to `web/dist`.

A live YouTube integration test is available separately:

```bash
go test -tags=integration ./internal/adapters/youtube -run TestLiveSearchTracksAgainstYouTube
```

Run it after local configuration and authorization. It uses your Google credentials and makes real API requests.

## Backend Docker image

Build from the repository root:

```bash
docker build -t playlistgo-api .
```

The included Compose file starts **MongoDB only**. The Dockerfile packages the Go API; the web interface is built and served separately.

Complete the first OAuth authorization on your local machine before a headless deployment. Then provide the token through a private, writable mounted file or directory and set `YOUTUBE_TOKEN_PATH` accordingly. Do not include credentials or tokens in the image. When MongoDB is another container, use its network hostname in `MONGODB_URI`, rather than `127.0.0.1`.

The current API has no user accounts or authentication for playlist endpoints, and playlists share one database. Treat it as a local development application; add access controls before exposing it to the internet.

## Project structure

| Path | Responsibility |
| --- | --- |
| `cmd/api` | API entry point |
| `internal/config` | Environment configuration |
| `internal/domain` | Track, search, and playlist models |
| `internal/app` | Application services |
| `internal/adapters/mongo` | MongoDB persistence and caching |
| `internal/adapters/youtube` | YouTube requests and OAuth |
| `internal/httpapi` | HTTP routes and JSON responses |
| `web/src` | React interface, API client, and local selection state |

## Troubleshooting

| Symptom | Check |
| --- | --- |
| API exits with a missing configuration error | Fill in `MONGODB_URI`, `CLIENT_ID`, and `CLIENT_SECRET`; run from the repository root |
| MongoDB connection fails | Start the Compose service and ensure the URI includes a database name |
| Startup waits without serving requests | Complete the Google authorization using the URL printed in the terminal |
| Google reports `redirect_uri_mismatch` | Use a Desktop OAuth client for the loopback callback |
| Google rejects your test account | Add it as a test user in the OAuth consent configuration |
| A new search returns HTTP 429 | The daily application budget or upstream quota may be exhausted; try a valid cached query or wait for the reset |
| Frontend cannot reach the API | Confirm the API is running on port 8080 and use the Vite development proxy |
| A separate frontend domain fails to call the API | Configure a same-origin reverse proxy or CORS at the proxy |

## Contributing

Ideas, bug reports, and focused pull requests are welcome. Include steps to reproduce issues and run the backend checks and frontend build before submitting a change.

The bundled Fugaz One, Lato, and Work Sans fonts are distributed under the SIL Open Font License; their license notices are included in `web/public/fonts`.
