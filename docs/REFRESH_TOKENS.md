# Refresh Token Implementation

## Overview

We have successfully implemented a comprehensive access/refresh token system for the charity application. This provides enhanced security and better user experience by allowing users to stay logged in longer while maintaining security through short-lived access tokens.

## Key Features

### 🔐 **Dual Token System**
- **Access Tokens**: Short-lived (15 minutes) for API authentication
- **Refresh Tokens**: Long-lived (7 days) for obtaining new access tokens

### 🗄️ **Database Storage**
- Refresh tokens are securely stored in the database
- Automatic cleanup of expired/revoked tokens
- Support for token revocation and logout

### 🔄 **Token Rotation**
- Each refresh operation generates new access AND refresh tokens
- Old refresh tokens are automatically revoked
- Prevents token replay attacks

## API Endpoints

### 1. **Login** - `POST /users/login`
**Enhanced Response:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "access_token_expires_at": "2025-07-24T09:15:00Z",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token_expires_at": "2025-07-31T08:30:00Z",
  "user": {
    "id": 1,
    "email": "user@example.com",
    "name": "John Doe",
    "balance": 1000000,
    "created_at": "2025-07-24T08:30:00Z"
  }
}
```

### 2. **Refresh Token** - `POST /auth/refresh`
**Request:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "access_token_expires_at": "2025-07-24T09:15:00Z",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token_expires_at": "2025-07-31T08:30:00Z"
}
```

### 3. **Logout** - `POST /auth/logout`
**Request:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response:**
```json
{
  "message": "successfully logged out"
}
```

### 4. **Logout All Devices** - `POST /auth/logout-all` (Protected)
**Headers:**
```
Authorization: Bearer <access_token>
```

**Response:**
```json
{
  "message": "successfully logged out from all devices"
}
```

## Database Schema

### Refresh Tokens Table
```sql
CREATE TABLE "refresh_tokens" (
  "id" uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  "user_id" bigint NOT NULL,
  "token_id" uuid NOT NULL UNIQUE,
  "expires_at" timestamptz NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "revoked_at" timestamptz
);

ALTER TABLE "refresh_tokens" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE;

CREATE INDEX ON "refresh_tokens" ("user_id");
CREATE INDEX ON "refresh_tokens" ("token_id");
CREATE INDEX ON "refresh_tokens" ("expires_at");
```

## Configuration

### Environment Variables
```env
# Token durations
ACCESS_TOKEN_DURATION=15m      # 15 minutes
REFRESH_TOKEN_DURATION=168h    # 7 days (168 hours)

# Token signing key (minimum 32 characters)
TOKEN_SYMMETRIC_KEY=your-secret-key-here
```

## Security Features

### 🛡️ **Token Validation**
- JWT signature verification
- Token type validation (access vs refresh)
- Expiration time checking
- Database existence verification for refresh tokens

### 🔒 **Token Revocation**
- Individual token revocation on logout
- Bulk revocation for all user devices
- Automatic cleanup of expired tokens

### 🔄 **Token Rotation**
- New tokens generated on each refresh
- Old refresh tokens immediately revoked
- Prevents token reuse attacks

## Implementation Details

### Token Types
```go
const (
    TokenTypeAccessToken  = 1
    TokenTypeRefreshToken = 2
)
```

### Token Payload
```go
type Payload struct {
    ID        uuid.UUID `json:"id"`
    Type      TokenType `json:"token_type"`
    UserID    int64     `json:"user_id"`
    IssuedAt  time.Time `json:"issued_at"`
    ExpiredAt time.Time `json:"expired_at"`
}
```

### Database Operations
- `CreateRefreshToken`: Store new refresh token
- `GetRefreshToken`: Retrieve active refresh token
- `RevokeRefreshToken`: Revoke specific token
- `RevokeAllUserRefreshTokens`: Revoke all user tokens
- `CleanupExpiredRefreshTokens`: Remove expired tokens

## Testing

### Comprehensive Test Coverage
- ✅ Token creation and validation
- ✅ Refresh token flow
- ✅ Token revocation
- ✅ Error handling
- ✅ Security edge cases

### Test Files
- `api/refresh_token_test.go` - Refresh token endpoint tests
- `token/jwt_maker_test.go` - Token creation/validation tests

## Usage Examples

### Frontend Integration
```javascript
// Store tokens after login
const loginResponse = await fetch('/users/login', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ email, password })
});

const { access_token, refresh_token } = await loginResponse.json();
localStorage.setItem('access_token', access_token);
localStorage.setItem('refresh_token', refresh_token);

// Refresh tokens when access token expires
const refreshTokens = async () => {
  const refreshToken = localStorage.getItem('refresh_token');
  const response = await fetch('/auth/refresh', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refresh_token: refreshToken })
  });
  
  const { access_token, refresh_token: newRefreshToken } = await response.json();
  localStorage.setItem('access_token', access_token);
  localStorage.setItem('refresh_token', newRefreshToken);
};

// Logout
const logout = async () => {
  const refreshToken = localStorage.getItem('refresh_token');
  await fetch('/auth/logout', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refresh_token: refreshToken })
  });
  
  localStorage.removeItem('access_token');
  localStorage.removeItem('refresh_token');
};
```

## Benefits

### 🚀 **Enhanced Security**
- Short-lived access tokens reduce exposure window
- Refresh tokens can be revoked immediately
- Token rotation prevents replay attacks

### 👥 **Better User Experience**
- Users stay logged in longer
- Seamless token refresh in background
- Multi-device support with individual logout

### 🔧 **Operational Benefits**
- Centralized token management
- Audit trail of token usage
- Easy revocation for security incidents

## Migration Notes

### Database Migration
Run the migration to add the refresh tokens table:
```bash
make migrateup
```

### Existing Clients
- Login endpoint now returns additional fields
- Existing access token validation unchanged
- New refresh endpoints are additive

## Monitoring & Maintenance

### Recommended Monitoring
- Track refresh token usage patterns
- Monitor failed refresh attempts
- Alert on unusual token revocation volumes

### Maintenance Tasks
- Regular cleanup of expired tokens
- Monitor token table growth
- Review token duration settings

---

## 🎉 Implementation Status: **COMPLETE**

The refresh token system is fully implemented, tested, and ready for production use!
