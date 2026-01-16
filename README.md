# EOG Alpaca Trading Platform

A fantasy trading platform that integrates with Alpaca's trading API, allowing users to view their portfolios, track performance, and engage with colleagues in a social trading environment.

## Features

- 🔐 API key authentication with Alpaca
- 📊 Real-time portfolio dashboard
- 📈 Historical performance charts
- 💼 Position tracking (stocks, crypto, options)
- 🏆 Leaderboard rankings
- 📱 Activity feed with social features
- 💬 Comments and reactions on trades

## Tech Stack

- **Backend:** Go 1.23+
- **Frontend:** HTMX, Templ, Tailwind CSS
- **Database:** SQLite3
- **APIs:** Alpaca Connect API & Trading API v2

## Getting Started

### Prerequisites

- Go 1.23 or higher
- Alpaca Paper Trading Account ([Sign up here](https://alpaca.markets/))
- Alpaca API Keys

### Installation

1. Clone the repository:
   ```bash
   cd /home/mshin/fantasy-trading
   ```

2. Get your Alpaca API Keys:
   - Create an Alpaca account at [alpaca.markets](https://alpaca.markets) if you don't have one
   - Log in to your Alpaca dashboard
   - Navigate to your [Paper Trading Dashboard](https://app.alpaca.markets/paper/dashboard/overview)
   - Find the **"Your API Keys"** section
   - Click **"View"** to see your API Key and Secret Key
   - **Note:** Keep these secure - you'll enter them when logging into the platform

3. Install dependencies:
   ```bash
   go mod download
   ```

4. Generate Templ templates:
   ```bash
   go install github.com/a-h/templ/cmd/templ@latest
   templ generate
   ```

5. Compile Tailwind CSS:
   ```bash
   ./tailwindcss -i ./static/css/input.css -o ./static/css/output.css --minify
   ```

6. Build and run the application:
   ```bash
   go build -o tmp/main .
   ./tmp/main
   ```

   Or use Air for hot reload during development:
   ```bash
   go install github.com/air-verse/air@latest
   air
   ```

7. Open your browser and navigate to:
   ```
   http://localhost:8080
   ```

8. Log in with your Alpaca API credentials:
   - Enter your API Key (starts with "PK...")
   - Enter your API Secret Key
   - Click "Login" to access your dashboard

## Development

### Project Structure

```
fantasy-trading/
├── internal/
│   ├── handlers/     # HTTP request handlers
│   ├── database/     # Database layer
│   ├── middleware/   # HTTP middleware
│   ├── alpaca/       # Alpaca API client
│   └── sync/         # Background sync jobs
├── templates/        # Templ templates
├── static/          # Static assets (CSS, JS)
├── data/            # SQLite database
└── main.go          # Application entry point
```

### Hot Reload

The project is configured with Air for hot reload during development:

```bash
air
```

This will watch for changes and automatically rebuild the application.

### Regenerating Templates

After modifying `.templ` files, regenerate the Go code:

```bash
templ generate
```

### Recompiling CSS

After modifying Tailwind classes:

```bash
./tailwindcss -i ./static/css/input.css -o ./static/css/output.css --minify
```

## Configuration

Environment variables can be set in the `.env` file:

- `PORT` - Server port (default: 8080)
- `DATABASE_PATH` - SQLite database file path (default: ./data/database.db)

**Note:** API keys are entered through the login page. Each user logs in with their own Alpaca API credentials.

## API Documentation

The application uses Alpaca's Trading API v2 with API key authentication:

- [Trading API Documentation](https://docs.alpaca.markets/docs/)
- [API Authentication](https://docs.alpaca.markets/docs/authentication)
- [Account API](https://docs.alpaca.markets/reference/getaccount-1)
- [Positions API](https://docs.alpaca.markets/reference/getallopenpositions)
- [Portfolio History](https://docs.alpaca.markets/reference/get-portfolio-history)

## Security

- API keys are stored securely in session database
- All sessions are secured with HttpOnly and SameSite cookies
- Content Security Policy headers
- API keys are only transmitted during login
- Each user manages their own API credentials
- No trading capability - read-only access to account data

## Next Steps

To complete the platform, implement:

1. **Leaderboard Page** - Rankings based on portfolio performance
2. **Activity Feed Page** - Platform-wide trade activity with reactions and comments
3. **Social Features** - Full commenting and reaction system
4. **Background Sync** - Periodic data sync from Alpaca API
5. **Docker Deployment** - Containerization and production deployment

## License

This is an internal EOG Resources project.
