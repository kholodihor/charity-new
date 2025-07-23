# Charity Application

A RESTful API for a charity platform built with Go, Gin, and PostgreSQL.

## Features

- **User Management**: Registration, authentication, and profile management
- **Charity Goals**: Create and manage fundraising goals
- **Donations**: Make donations to charity goals with transaction tracking
- **Events**: Create and manage charity events
- **Event Bookings**: Book and manage event attendance

## Tech Stack

- **Backend**: Go with Gin framework
- **Database**: PostgreSQL with SQLC for type-safe queries
- **Authentication**: JWT tokens
- **Configuration**: Viper for environment-based config

## Project Structure

```
├── api/                 # HTTP handlers and routes
│   ├── server.go       # Server setup and routing
│   ├── user.go         # User management endpoints
│   ├── goal.go         # Goal management endpoints
│   ├── donation.go     # Donation endpoints
│   ├── event.go        # Event management endpoints
│   └── middleware.go   # Authentication middleware
├── db/
│   ├── migration/      # Database migrations
│   ├── query/          # SQL queries
│   └── sqlc/          # Generated Go code from SQL
├── token/              # JWT token management
├── util/               # Utility functions and config
├── main.go            # Application entry point
└── app.env            # Environment configuration
```

## Setup

1. **Prerequisites**
   - Go 1.21+
   - PostgreSQL
   - SQLC (for code generation)

2. **Database Setup**
   ```bash
   createdb charity
   migrate -path db/migration -database "postgresql://postgres:postgres@localhost:5432/charity?sslmode=disable" -verbose up
   ```

3. **Configuration**
   Update `app.env` with your database connection and other settings:
   ```env
   DB_DRIVER=postgres
   DB_SOURCE=postgresql://postgres:postgres@localhost:5432/charity?sslmode=disable
   SERVER_ADDRESS=0.0.0.0:8080
   TOKEN_SYMMETRIC_KEY=your-32-character-secret-key
   ACCESS_TOKEN_DURATION=15m
   ```

4. **Run the Application**
   ```bash
   go run main.go
   ```

## API Endpoints

### Public Endpoints
- `POST /users` - Register a new user
- `POST /users/login` - User login
- `GET /goals` - List charity goals
- `GET /goals/:id` - Get specific goal
- `GET /events` - List events
- `GET /events/:id` - Get specific event
- `GET /donations` - List donations
- `GET /users` - List users

### Protected Endpoints (Require Authentication)
- `GET /users/me` - Get current user profile
- `PUT /users/me` - Update current user profile
- `POST /goals` - Create new goal
- `PUT /goals/:id` - Update goal
- `DELETE /goals/:id` - Delete goal
- `POST /donations` - Make a donation
- `POST /events` - Create new event
- `PUT /events/:id` - Update event
- `DELETE /events/:id` - Delete event
- `POST /events/:id/book` - Book an event
- `DELETE /events/:id/book` - Cancel event booking

## Authentication

Include the JWT token in the Authorization header:
```
Authorization: Bearer <your_jwt_token>
```

## Database Schema

- **users**: User accounts with email, name, balance
- **goals**: Charity fundraising goals
- **donations**: Donation transactions
- **events**: Charity events
- **event_bookings**: Event attendance tracking

## Development

### Generate SQLC Code
```bash
sqlc generate
```

### Run Tests
```bash
go test ./...
```

### Build
```bash
go build .
```

## API Documentation

See [API_ENDPOINTS.md](API_ENDPOINTS.md) for detailed API documentation with request/response examples.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the MIT License.
