# Transit API JP

Go REST API for Transit (trains/buses) in Japan

Uses NAVITIME API via RAPIDAPI

## Running Locally with Docker Compose

1. **Ensure you have Docker and Docker Compose installed.**

2. **Create a `.env` file** in the project root with your required environment variables (e.g., RAPIDAPI_KEY, RAPIDAPI_TRANSPORT_HOST, RAPIDAPI_TRANSIT_HOST).

3. **Build and run the service:**

   ```sh
   docker compose up --build
   ```

4. **Access the API** at [http://localhost:8080](http://localhost:8080)

## API Documentation (Swagger)

This API includes comprehensive Swagger/OpenAPI documentation for easy exploration and testing.

### Accessing Swagger UI

Once the server is running, you can access the interactive API documentation at:

- **Swagger UI**: [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)
- **JSON Specification**: [http://localhost:8080/swagger/doc.json](http://localhost:8080/swagger/doc.json)

### Generating Documentation

If you need to regenerate the Swagger documentation after making changes to the API:

1. **Install swag CLI** (if not already installed):
   ```sh
   go install github.com/swaggo/swag/cmd/swag@latest
   ```

2. **Generate docs** from the project root:
   ```sh
   swag init
   ```

This will create/update the `docs/` folder with the latest API documentation.

### Features

- **Interactive Testing**: Test API endpoints directly from the browser
- **Request/Response Examples**: See example payloads for all endpoints
- **Model Documentation**: Complete documentation of all data structures
- **Parameter Validation**: Clear parameter requirements and constraints
- **Multi-language Support**: Documentation includes both Japanese and English examples

## Type System

This API uses strongly-typed Go structs instead of `interface{}` (any) types for better type safety, performance, and maintainability. All API responses are properly structured using the types defined in `/model/transit.go`.

### Key Type Structures

- **`TransitResponse`** - Top-level response containing transit route options and units
- **`TransitItem`** - Individual route option with summary and detailed sections
- **`Section`** - Route segments that can be either points (stations) or moves (transportation)
- **`Transport`** - Detailed transportation information including fares, companies, and links
- **`AutocompleteResponse`** - Station search results with filtering capabilities

### Benefits

- **Type Safety**: Compile-time error detection for field access and type mismatches
- **IDE Support**: Full autocomplete, go-to-definition, and refactoring capabilities
- **Performance**: Eliminates runtime type assertions and reflection overhead
- **Maintainability**: Clear data structure contracts and easier debugging

## Translation System

For English responses (`lang=en`), the API automatically translates Japanese station names, company names, and line names from Kanji/Kana to Romaji using the `github.com/jtclarkjr/kanjikana` package. The translation system recursively processes all relevant text fields in the response structure.

## Transit

Calls go transit enpoint with start, goal and start date. Return all data from those two point.

### JA

`/transit?start={station_name}&goal={station_name}&start_time={start_time} (datetime format: 2020-08-19T10%3A00%3A00)`

### EN

Need to add `lang=en` param to get english names

`/transit?lang=en&start={station_name}&goal={station_name}&start_time={start_time} (datetime format: 2020-08-19T10%3A00%3A00)`

Tokenize the kanji to kana then convert the kana to romaji

### Response Structure

The transit API returns a `TransitResponse` containing:

- **Items**: Array of route options with summaries and detailed sections
- **Unit**: Measurement units used in the response (currency, distance, time, etc.)

Each route item includes:

- **Summary**: Overview with start/goal points, transit count, fare, and timing
- **Sections**: Detailed step-by-step route segments including stations and transportation details

## Autocomplete

Returns a list of objects for stations based on input using `word` param

### JA

`/autocomplete?word=station_name`

### EN

Need to add `lang=en` param to get english names

`/autocomplete?lang=en&word=station_name`

Tokenize the kanji to kana then convert the kana to romaji

### Response Structure

The autocomplete API returns a `FilteredAutocompleteResponse` containing:

- **Items**: Array of filtered stations (only stations, not other transport nodes)

Each station includes:

- **ID**: Unique station identifier
- **Name**: Station name (translated to Romaji if `lang=en`)
- **Type**: Always "station" (other node types are filtered out)
