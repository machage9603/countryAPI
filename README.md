# Country API

## Overview

This is a RESTful API developed in Go using Gin for routing, GORM for ORM, and PostgreSQL for data storage. It integrates with external APIs to fetch country information (from [RestCountries](https://restcountries.com)) and exchange rates (from [Open Exchange Rates](https://open.er-api.com)). The API caches this data, computes an estimated GDP per country, generates a summary PNG image, and exposes endpoints for data management and querying.

Key features:
- Refresh data from external sources and cache in DB.
- Filter and sort countries (e.g., by region, currency, GDP, population).
- Handle cases where countries lack currencies or exchange rates.
- Generate and serve a visual summary image.
- Error handling for external API failures and validation.

## Prerequisites

- Go (version 1.21 or later recommended).
- PostgreSQL database (local or hosted).
- Docker (optional, for easy local DB setup).
- Git (for cloning the repository).
- Tools like Postman or curl for testing API endpoints.

## Installation

1. **Clone the Repository**:

2. **Install Dependencies**:
Run the following to download Go modules:

3. **Set Up Environment Variables**:
Create a `.env` file in the root directory with the following content:
- Replace the `DATABASE_URL` with your PostgreSQL connection string.
- For a local setup, see the "Local Database Setup" section below.

## Local Database Setup (Using Docker)

If you don't have PostgreSQL installed locally, use Docker to spin up a container:

1. Pull and run the PostgreSQL Docker image:

2. Verify the container is running:

3. (Optional) Connect to the DB using a client like pgAdmin or psql:

The database will be accessible at `localhost:5432` with username `postgres` and password `password`.

## Running the Application

1. Start the server:

The database will be accessible at `localhost:5432` with username `postgres` and password `password`.

## Running the Application

1. Start the server:
The API will be available at `http://localhost:8080` (or the port specified in `.env`).

2. (Optional) Build an executable:

## API Endpoints

All responses are in JSON format unless specified.

- **POST /countries/refresh**:
- Fetches fresh data from external APIs, updates/inserts into DB, computes estimated GDP, and generates a summary image.
- Response: `{ "message": "Countries refreshed successfully", "last_refreshed_at": "2025-10-28T12:00:00Z" }`
- Errors: 503 if external APIs fail.

- **GET /countries**:
- Retrieves all countries from DB.
- Query params:
 - `region`: Filter by region (e.g., `?region=Africa`).
 - `currency`: Filter by currency code (e.g., `?currency=NGN`).
 - `sort`: Sort by `gdp_desc`, `gdp_asc`, `population_desc`, `population_asc` (default: name ASC).
- Response: Array of country objects (see sample below).

- **GET /countries/:name**:
- Retrieves a single country by name (case-insensitive).
- Response: Country object.
- Errors: 404 if not found.

- **DELETE /countries/:name**:
- Deletes a country by name (case-insensitive).
- Response: `{ "message": "Country deleted successfully" }`
- Errors: 404 if not found.

- **GET /status**:
- Shows total countries and last refresh timestamp.
- Response: `{ "total_countries": 250, "last_refreshed_at": "2025-10-28T12:00:00Z" }`

- **GET /countries/image**:
- Serves the generated summary PNG image (from `cache/summary.png`).
- Response: Image file (binary).
- Errors: 404 if image not found (run refresh first).

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

