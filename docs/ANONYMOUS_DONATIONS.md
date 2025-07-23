# Anonymous Donations Feature

## Overview

The charity application now supports **anonymous donations** - allowing users to donate to charity goals without requiring registration or authentication. This feature enables broader participation in charitable giving by removing barriers for users who prefer to donate anonymously.

## Key Features

### 🎯 **No Registration Required**
- Users can donate without creating an account
- No authentication or login needed
- Immediate donation capability

### 💰 **External Payment Processing**
- Anonymous donations don't deduct from user balance
- Assumes external payment processing (Stripe, PayPal, etc.)
- Real money transactions handled outside the system

### 🔒 **Privacy Focused**
- All anonymous donations are marked as `is_anonymous: true`
- No user ID associated with the donation
- Maintains donor privacy

### ✅ **Goal Validation**
- Validates that the target goal exists and is active
- Updates goal's collected amount automatically
- Maintains data integrity

## API Endpoint

### POST `/donations/anonymous`

**Description:** Create an anonymous donation to a charity goal

**Authentication:** None required (public endpoint)

**Request Body:**
```json
{
  "goal_id": 123,
  "amount": 5000
}
```

**Response (201 Created):**
```json
{
  "id": 456,
  "user_id": null,
  "goal_id": 123,
  "amount": 5000,
  "is_anonymous": true,
  "created_at": "2023-01-01T12:00:00Z"
}
```

**Error Responses:**
- `400 Bad Request` - Invalid request data (negative amount, missing fields)
- `404 Not Found` - Goal doesn't exist
- `500 Internal Server Error` - Database or server error

## Database Schema

The existing schema already supports anonymous donations:

```sql
CREATE TABLE "donations" (
  "id" bigserial PRIMARY KEY,
  "user_id" bigint,                    -- Nullable for anonymous donations
  "goal_id" bigint NOT NULL,
  "amount" bigint NOT NULL,
  "is_anonymous" boolean NOT NULL DEFAULT false,
  "created_at" timestamptz NOT NULL DEFAULT 'now()'
);
```

**Key Points:**
- `user_id` is nullable - allows anonymous donations
- `is_anonymous` flag distinguishes anonymous vs registered user donations
- Foreign key constraint on `user_id` allows NULL values

## Implementation Details

### Transaction Logic

The `DonateToGoalTx` transaction handles both registered and anonymous donations:

```go
// Anonymous donation parameters
arg := db.DonateToGoalTxParams{
    GoalID: req.GoalID,
    UserID: pgtype.Int8{
        Valid: false, // No user ID for anonymous donations
    },
    Amount:      req.Amount,
    IsAnonymous: true, // Always true for anonymous donations
}
```

### Key Differences from Registered Donations

| Feature | Registered Donation | Anonymous Donation |
|---------|-------------------|-------------------|
| Authentication | Required (JWT token) | None |
| User Balance | Deducted from balance | No balance check |
| User ID | Set from token | NULL |
| Anonymous Flag | User choice | Always true |
| Endpoint | `/donations` | `/donations/anonymous` |

## Usage Examples

### Successful Anonymous Donation
```bash
curl -X POST http://localhost:8080/donations/anonymous \
  -H "Content-Type: application/json" \
  -d '{
    "goal_id": 1,
    "amount": 2500
  }'
```

### Response
```json
{
  "id": 15,
  "user_id": null,
  "goal_id": 1,
  "amount": 2500,
  "is_anonymous": true,
  "created_at": "2023-01-01T12:00:00Z"
}
```

## Testing

Comprehensive tests are included in `anonymous_donation_test.go`:

- ✅ **Successful donation** - Valid goal and amount
- ✅ **Goal not found** - Invalid goal ID
- ✅ **Internal error** - Database failures
- ✅ **Invalid data** - Negative amounts, missing fields

Run tests:
```bash
go test -v ./api -run TestCreateAnonymousDonationAPI
```

## Security Considerations

### ✅ **Implemented Safeguards**
- Input validation (positive amounts, required fields)
- Goal existence validation
- SQL injection protection via SQLC
- Transaction integrity

### ⚠️ **External Requirements**
- **Payment Processing**: Integrate with external payment gateway
- **Fraud Prevention**: Implement rate limiting and fraud detection
- **Amount Limits**: Consider maximum donation limits
- **Audit Trail**: Log anonymous donations for compliance

## Integration Points

### Payment Gateway Integration
```go
// Example integration point
func (server *Server) createAnonymousDonation(ctx *gin.Context) {
    // ... validation code ...
    
    // Process payment with external gateway
    paymentResult, err := server.paymentGateway.ProcessPayment(PaymentRequest{
        Amount:      req.Amount,
        Description: fmt.Sprintf("Donation to goal %d", req.GoalID),
        // ... other payment details
    })
    
    if err != nil {
        ctx.JSON(http.StatusPaymentRequired, gin.H{"error": "payment failed"})
        return
    }
    
    // Create donation record after successful payment
    // ... existing donation creation code ...
}
```

## Benefits

### 🌟 **For Donors**
- **Convenience**: No registration barriers
- **Privacy**: Complete anonymity
- **Speed**: Immediate donation capability
- **Accessibility**: Lower barrier to entry

### 🌟 **For Charity**
- **Increased Donations**: More users can participate
- **Broader Reach**: Appeals to privacy-conscious donors
- **Simplified Process**: Streamlined donation flow
- **Higher Conversion**: Reduced friction

## Future Enhancements

### 🚀 **Potential Improvements**
1. **Payment Gateway Integration** - Stripe, PayPal, etc.
2. **Rate Limiting** - Prevent spam donations
3. **Donation Receipts** - Email receipts without registration
4. **Recurring Donations** - Anonymous subscription support
5. **Donation Matching** - Corporate matching programs
6. **Analytics** - Anonymous donation tracking and reporting

## Conclusion

The anonymous donation feature significantly enhances the charity platform by:
- Removing barriers to charitable giving
- Maintaining donor privacy
- Increasing potential donation volume
- Providing a seamless user experience

The implementation is secure, well-tested, and ready for production use with proper payment gateway integration.
