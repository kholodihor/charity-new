package util

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5/pgtype"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz"

// RandomInt generates a random integer between min and max
func RandomInt(min, max int64) int64 {
	n, _ := rand.Int(rand.Reader, big.NewInt(max-min+1))
	return n.Int64() + min
}

// RandomString generates a random string of length n
func RandomString(n int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz"
	var sb strings.Builder
	sb.Grow(n)
	k := len(alphabet)

	for i := 0; i < n; i++ {
		// Use crypto/rand for secure random number generation
		randInt, err := rand.Int(rand.Reader, big.NewInt(int64(k)))
		if err != nil {
			// Fallback to a default character if there's an error
			sb.WriteByte('a')
			continue
		}
		c := alphabet[randInt.Int64()]
		sb.WriteByte(c)
	}

	return sb.String()
}

// RandomOwner generates a random owner name
func RandomOwner() string {
	return RandomString(6)
}

// RandomMoney generates a random amount of money
func RandomMoney() int64 {
	return RandomInt(0, 1000)
}

// RandomCurrency generates a random currency code
func RandomCurrency() string {
	currencies := []string{"USD", "EUR", "UAH"}
	n := len(currencies)
	randInt, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return currencies[0] // Return first currency as fallback
	}
	return currencies[randInt.Int64()]
}

// RandomEmail generates a random email
func RandomEmail() string {
	return fmt.Sprintf("%s@email.com", RandomString(6))
}

// RandomName generates a random name with first letter capitalized
func RandomName() string {
	s := RandomString(6)
	if len(s) > 0 {
		r := []rune(s)
		r[0] = unicode.ToUpper(r[0])
		s = string(r)
	}
	return s
}

// RandomUserParams generates random user creation parameters
func RandomUserParams() (email string, name pgtype.Text) {
	return RandomEmail(), pgtype.Text{String: RandomName() + " " + RandomName(), Valid: true}
}

// RandomGoalParams generates random goal creation parameters
func RandomGoalParams() (title string, description pgtype.Text, targetAmount pgtype.Int8, isActive bool) {
	return "Goal " + RandomString(6),
		pgtype.Text{String: "Description for " + RandomString(12), Valid: true},
		pgtype.Int8{Int64: RandomInt(1000, 100000), Valid: true},
		true
}

// RandomDonationParams generates random donation creation parameters
func RandomDonationParams(userID, goalID int64) (donorID pgtype.Int8, amount int64, isAnonymous bool) {
	return pgtype.Int8{Int64: userID, Valid: userID > 0},
		RandomInt(100, 10000), // $1.00 to $100.00
		false
}

// RandomEventParams generates random event creation parameters
func RandomEventParams() (name string, place string, date time.Time) {
	return "Event " + RandomString(8),
		"Venue " + RandomString(6) + ", " + RandomString(8) + " City",
		time.Now().Add(time.Duration(RandomInt(1, 365)) * 24 * time.Hour) // 1-365 days from now
}
