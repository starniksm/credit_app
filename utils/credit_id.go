package utils

import (
	"fmt"
	"math/rand"
	"time"
)

// CreditTypeMap maps short codes to full credit types
var CreditTypeMap = map[string]string{
	"AK": "АК", // Автокредит
	"PK": "ПК", // Потребительский кредит
	"IP": "ИП", // Ипотека
	"KK": "КК", // Кредитная карта
	"BK": "БК", // Бизнес-кредит
}

// GenerateCreditApplicationID generates a credit application ID in the format: TT-XXXXXXXX-PP-NN
// TT (2 chars) - Credit type in Latin
// XXXXXXXX - Client ID (8 digits)
// PP - Credit product number (2 digits)
// NN - Sequence number (2 digits)
func GenerateCreditApplicationID(creditType string, clientID uint, productNum string, sequenceNum string) string {
	// Pad clientID to 8 digits
	clientIDStr := fmt.Sprintf("%08d", clientID)

	// Validate credit type - use Latin codes only for URL safety
	shortType := creditType
	if shortType == "" {
		shortType = "PK" // Default to consumer loan
	}

	// Map to Latin code (always use Latin for URL safety)
	latinType := shortType
	switch shortType {
	case "АК", "AK":
		latinType = "AK"
	case "ПК", "PK":
		latinType = "PK"
	case "ИП", "IP":
		latinType = "IP"
	case "КК", "KK":
		latinType = "KK"
	case "БК", "BK":
		latinType = "BK"
	default:
		latinType = "PK"
	}

	// Validate product number (ensure it's 2 digits)
	productNumPadded := fmt.Sprintf("%02s", productNum)
	if len(productNumPadded) > 2 {
		productNumPadded = productNumPadded[:2]
	}

	// Validate sequence number (ensure it's 2 digits)
	sequenceNumPadded := fmt.Sprintf("%02s", sequenceNum)
	if len(sequenceNumPadded) > 2 {
		sequenceNumPadded = sequenceNumPadded[:2]
	}

	return fmt.Sprintf("%s-%s-%s-%s", latinType, clientIDStr, productNumPadded, sequenceNumPadded)
}

// GenerateRandomCreditApplicationID generates a random credit application ID for testing
func GenerateRandomCreditApplicationID(creditType string) string {
	rand.Seed(time.Now().UnixNano())

	clientID := rand.Intn(99999999)                     // Random 8-digit client ID
	productNum := fmt.Sprintf("%02d", rand.Intn(99)+1)  // Random 2-digit product number
	sequenceNum := fmt.Sprintf("%02d", rand.Intn(99)+1) // Random 2-digit sequence number

	return GenerateCreditApplicationID(creditType, uint(clientID), productNum, sequenceNum)
}

// ParseCreditApplicationID parses a credit application ID and returns its components
func ParseCreditApplicationID(id string) (creditType, clientID, productNum, sequenceNum string) {
	// Format: TT-XXXXXXXX-PP-NN
	// Example: AK-12345678-03-01
	var parts []string
	currentPart := ""

	for _, char := range id {
		if char == '-' {
			parts = append(parts, currentPart)
			currentPart = ""
		} else {
			currentPart += string(char)
		}
	}
	parts = append(parts, currentPart)

	if len(parts) >= 4 {
		return parts[0], parts[1], parts[2], parts[3]
	}

	return "", "", "", ""
}

// GetCreditTypeText returns the human-readable credit type name
func GetCreditTypeText(shortType string) string {
	if text, exists := CreditTypeMap[shortType]; exists {
		return text
	}
	return shortType // Return as-is if not found
}

// GenerateCreditID generates a unique credit application ID
func GenerateCreditID() string {
	return GenerateRandomCreditApplicationID("PK")
}
