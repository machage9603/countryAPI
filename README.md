# Country API

## Overview

This is a RESTful API developed in Go using the Gin framework for routing, GORM for object-relational mapping (ORM), and PostgreSQL for persistent data storage. The API integrates with two external services: [RestCountries](https://restcountries.com) for fetching country details (such as name, capital, region, population, flag, and currencies) and [Open Exchange Rates](https://open.er-api.com) for retrieving USD-based exchange rates. It processes this data by caching it in the database, computing an estimated GDP for each country using the formula `population × random(1000–2000) ÷ exchange_rate`, and generating a visual summary image in PNG format.

The API is designed for data aggregation, caching, and visualization tasks. It supports CRUD-like operations (refresh/create/update, read, delete) on country records, with built-in filters, sorting, and error handling. Special cases are handled gracefully, such as countries without currencies (set `estimated_gdp` to 0) or missing exchange rates (set to null). The summary image is regenerated on each refresh and served via an endpoint.

Key features:
- **Data Refresh**: Fetches and caches data from external APIs on demand.
- **Querying with Filters and Sorting**: Supports region, currency filters, and sorting by GDP or population.
- **Special Handling**: Manages edge cases for currencies and rates without disrupting storage.
- **Image Generation**: Creates a PNG summary with total countries, top 5 by GDP, and refresh timestamp.
- **Error Handling**: Returns appropriate HTTP status codes (e.g., 503 for external API failures, 404 for not found) with JSON error messages.
- **Persistence**: Uses PostgreSQL for reliable data storage across restarts.

This project is suitable for learning about API development, database integration, external API consumption, and image generation in Go.

## Prerequisites

- **Go**: Version 1.21 or later (check with `go version`; install from [golang.org](https://golang.org/dl/)).
- **PostgreSQL Database**: A running instance (local or hosted).
- **Docker**: Optional, but recommended for easy local PostgreSQL setup (install from [docker.com](https://www.docker.com/get-started/)).
- **Git**: For cloning the repository.
- **Testing Tools**: Postman, curl, or similar for API testing.
- **Optional**: A code editor like VS Code for viewing/editing files.

## Installation

1. **Clone the Repository**:
   ```
   git clone https://github.com/yourusername/country-api.git
   cd country-api
   ```
   Replace `yourusername` with your GitHub username or the actual repository URL.

2. **Install Dependencies**:
   Run the following to download and tidy Go modules based on `go.mod`:
   ```
   go mod tidy
   go mod download
   ```

3. **Set Up Environment Variables**:
   Create a `.env` file in the root directory with the following content:
   ```
   DATABASE_URL=postgres://postgres:password@localhost:5432/postgres?sslmode=disable
   PORT=8080  # Optional; defaults to 8080 if not set
   ```
   - Replace the `DATABASE_URL` with your actual PostgreSQL connection string (e.g., include your password and database name).
   - For a local setup, see the "Local Database Setup" section below.
   - The app uses [godotenv](https://github.com/joho/godotenv) to load these variables.

## Local Database Setup (Using Docker)

If you don't have PostgreSQL installed locally, use Docker to quickly set up a containerized instance:

1. Pull and run the PostgreSQL Docker image:
   ```
   docker run --name country-db -e POSTGRES_PASSWORD=password -p 5432:5432 -d postgres
   ```
   - This starts a PostgreSQL server with default user `postgres` and the specified password.
   - If port 5432 is in use, change it (e.g., `-p 5433:5432`) and update your `.env` accordingly.

2. Verify the container is running:
   ```
   docker ps
   ```
   - Look for `country-db` in the list. If issues arise, check logs with `docker logs country-db`.

3. (Optional) Connect to the DB using a client like pgAdmin (download from [pgadmin.org](https://www.pgadmin.org/)) or psql:
   ```
   docker exec -it country-db psql -U postgres
   ```
   - In psql, you can run commands like `\l` to list databases or create tables manually if needed.

The database will be accessible at `localhost:5432` with username `postgres` and password `password`. The app will auto-migrate the schema on startup using GORM.

## Running the Application

1. Start the server:
   ```
   go run main.go
   ```
   - The API will be available at `http://localhost:8080` (or the port specified in `.env`).
   - On first run, it connects to the database and migrates the `Country` model schema.

2. (Optional) Build an executable for easier deployment or running:
   ```
   go build -o country-api
   ./country-api  # On Windows: country-api.exe
   ```

The server runs in debug mode by default (using Gin's default settings). For production, consider setting `GIN_MODE=release` in `.env`.

## API Endpoints

All responses are in JSON format unless specified (e.g., the image endpoint returns binary data).

- **POST /countries/refresh**:
  - Fetches fresh data from external APIs, updates/inserts into DB, computes estimated GDP, and generates a summary image.
  - No request body required.
  - Response: `{ "message": "Countries refreshed successfully", "last_refreshed_at": "2025-10-28T12:00:00Z" }`
  - Errors: 503 if external APIs fail (e.g., `{ "error": "External data source unavailable", "details": "Could not fetch data from restcountries.com" }`).

- **GET /countries**:
  - Retrieves all countries from the DB.
  - Query params:
    - `region`: Filter by region (e.g., `?region=Africa`).
    - `currency`: Filter by currency code (e.g., `?currency=NGN`).
    - `sort`: Sort by `gdp_desc`, `gdp_asc`, `population_desc`, `population_asc` (default: name ASC).
  - Response: Array of country objects (see sample below).

- **GET /countries/:name**:
  - Retrieves a single country by name (case-insensitive).
  - Response: Country object.
  - Errors: 404 if not found (e.g., `{ "error": "Country not found" }`).

- **DELETE /countries/:name**:
  - Deletes a country by name (case-insensitive).
  - Response: `{ "message": "Country deleted successfully" }`
  - Errors: 404 if not found.

- **GET /status**:
  - Shows total countries and last refresh timestamp.
  - Response: `{ "total_countries": 250, "last_refreshed_at": "2025-10-28T12:00:00Z" }`

- **GET /countries/image**:
  - Serves the generated summary PNG image (from `cache/summary.png`).
  - Response: Image file (binary; set `Content-Type: image/png` in client if needed).
  - Errors: 404 if image not found (run refresh first; e.g., `{ "error": "Summary image not found" }`).

### Sample Country Object
```json
{
  "id": 1,
  "name": "Nigeria",
  "capital": "Abuja",
  "region": "Africa",
  "population": 206139589,
  "currency_code": "NGN",
  "exchange_rate": 1600.23,
  "estimated_gdp": 25767448125.2,
  "flag_url": "https://flagcdn.com/ng.svg",
  "last_refreshed_at": "2025-10-28T12:00:00Z"
}
```

## Testing

Test the API using curl or Postman after running the server:

1. Refresh data:
   ```
   curl -X POST http://localhost:8080/countries/refresh
   ```

2. Get all countries:
   ```
   curl http://localhost:8080/countries
   ```

3. Get filtered/sorted countries:
   ```
   curl "http://localhost:8080/countries?region=Africa&sort=gdp_desc"
   ```

4. Get a single country:
   ```
   curl http://localhost:8080/countries/Nigeria
   ```

5. Delete a country:
   ```
   curl -X DELETE http://localhost:8080/countries/Nigeria
   ```

6. Get status:
   ```
   curl http://localhost:8080/status
   ```

7. Get and save the image:
   ```
   curl http://localhost:8080/countries/image --output summary.png
   ```

- **Verification Tips**: After refresh, check countries like "Antarctica" (`curl http://localhost:8080/countries/Antarctica`) for `"estimated_gdp": 0`. Inspect the DB (using pgAdmin) to confirm records. View `cache/summary.png` for the generated image.

## Deployment

For hosting (e.g., on Railway, Heroku, AWS, or similar):

1. Push to GitHub (if not already): `git add .`, `git commit -m "Initial commit"`, `git push`.
2. On Railway (recommended for simplicity):
   - Create an account at [railway.app](https://railway.app).
   - New project > Add PostgreSQL service (Railway provides `DATABASE_URL`).
   - Deploy from GitHub repo.
   - Set env vars: Reference the DB's `DATABASE_URL`.
   - Railway auto-builds and deploys; access at the provided URL.
3. Notes: Storage for `cache/summary.png` is ephemeral on some hosts—refresh regenerates it. For production, consider cloud storage for images.

## Troubleshooting

- **DB Connection Failed**: Verify Docker container is running (`docker ps`), password matches `.env`, and port is free. Test connection with psql.
- **External API Errors**: If 503 on refresh, check internet or API status (e.g., via browser: https://restcountries.com/v2/all).
- **Image Generation Failed**: Check logs for font errors; ensure `cache` directory exists and is writable.
- **Sorting with NULLs**: GDP sorts handle nulls (desc: nulls last; asc: nulls first).
- **General Errors**: Run with debug logs; check console output. For 500 errors, add more logging in code if needed.
- **Go Version Issues**: Ensure compatible version; run `go version`.

If issues persist, check Go docs or open an issue on the repo.

## Dependencies

- [Gin](https://github.com/gin-gonic/gin): HTTP web framework.
- [GORM](https://gorm.io): ORM library with PostgreSQL driver.
- [godotenv](https://github.com/joho/godotenv): Loads .env files.
- [golang/freetype](https://github.com/golang/freetype): For text rendering in images.
- Full list: See `go.mod` for versions and indirect dependencies.

To update: Run `go get -u` for packages.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details (create one if missing with standard MIT text).