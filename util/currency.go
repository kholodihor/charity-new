package util

// Constants for all supported currencies
const (
	USD = "USD"
	EUR = "EUR"
	UAH = "UAH"
)

// IsSupportedCurrency returns true if the currency is supported
func IsSupportedCurrency(currency string) bool {
	switch currency {
	case USD, EUR, UAH:
		return true
	}
	return false
}
