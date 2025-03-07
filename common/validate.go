package common

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/initia-labs/weave/crypto"
)

var (
	// Denominations can be 3 ~ 128 characters long and support letters, followed by either
	// a letter, a number or a separator ('/', ':', '.', '_' or '-').
	reDnmString = `[a-zA-Z][a-zA-Z0-9/:._-]{2,127}`
	reDecAmt    = `[[:digit:]]+(?:\.[[:digit:]]+)?|\.[[:digit:]]+`
	reSpc       = `[[:space:]]*`
	reDnm       *regexp.Regexp
	reDecCoin   *regexp.Regexp

	MaxBitLen = 256

	// LegacyPrecision number of decimal places
	LegacyPrecision = 18

	// LegacyDecimalPrecisionBits bits required to represent the above precision
	// Ceiling[Log2[10^Precision - 1]]
	LegacyDecimalPrecisionBits = 60

	// decimalTruncateBits is the minimum number of bits removed
	// by a truncate operation. It is equal to
	// Floor[Log2[10^Precision - 1]].
	decimalTruncateBits = LegacyDecimalPrecisionBits - 1

	maxDecBitLen = MaxBitLen + decimalTruncateBits

	ErrLegacyEmptyDecimalStr      = errors.New("decimal string cannot be empty")
	ErrLegacyInvalidDecimalLength = errors.New("invalid decimal length")
	ErrLegacyInvalidDecimalStr    = errors.New("invalid decimal string")
)

func init() {
	SetCoinDenomRegex(DefaultCoinDenomRegex)
}

// DefaultCoinDenomRegex returns the default regex string
func DefaultCoinDenomRegex() string {
	return reDnmString
}

// coinDenomRegex returns the current regex string and can be overwritten for custom validation
var coinDenomRegex = DefaultCoinDenomRegex

// SetCoinDenomRegex allows for coin's custom validation by overriding the regular
// expression string used for denom validation.
func SetCoinDenomRegex(reFn func() string) {
	coinDenomRegex = reFn

	reDnm = regexp.MustCompile(fmt.Sprintf(`^%s$`, coinDenomRegex()))
	reDecCoin = regexp.MustCompile(fmt.Sprintf(`^(%s)%s(%s)$`, reDecAmt, reSpc, coinDenomRegex()))
}

// ValidateDenom is the default validation function for Coin.Denom.
func ValidateDenom(denom string) error {
	if !reDnm.MatchString(denom) {
		return fmt.Errorf("invalid denom: %s", denom)
	}
	return nil
}

func ValidateDenomWithReserved(reservedDenoms []string) func(denom string) error {
	return func(denom string) error {
		if !reDnm.MatchString(denom) {
			return fmt.Errorf("invalid denom: %s", denom)
		}

		for _, reserved := range reservedDenoms {
			if denom == reserved {
				return fmt.Errorf("denom %s is reserved and cannot be used (reserved denom for this rollup: %v)", denom, reservedDenoms)
			}
		}

		return nil
	}
}

func NoOps(_ string) error {
	return nil
}

func ValidateDecCoin(coinStr string) (err error) {
	coinStr = strings.TrimSpace(coinStr)

	matches := reDecCoin.FindStringSubmatch(coinStr)
	if matches == nil {
		return fmt.Errorf("invalid decimal coin expression: %s", coinStr)
	}

	amountStr, denomStr := matches[1], matches[2]

	err = ValidateDecFromStr(amountStr)
	if err != nil {
		return fmt.Errorf("failed to parse decimal coin amount: %s", amountStr)
	}

	if err := ValidateDenom(denomStr); err != nil {
		return fmt.Errorf("invalid denom cannot contain spaces: %s", err)
	}

	return nil
}

func ValidateDecFromStr(str string) error {
	if str[0] == '-' {
		return fmt.Errorf("decimal string cannot be positive")
	}

	if len(str) == 0 {
		return ErrLegacyEmptyDecimalStr
	}

	strs := strings.Split(str, ".")
	lenDecs := 0
	combinedStr := strs[0]

	if len(strs) == 2 { // has a decimal place
		lenDecs = len(strs[1])
		if lenDecs == 0 || len(combinedStr) == 0 {
			return ErrLegacyInvalidDecimalLength
		}
		combinedStr += strs[1]
	} else if len(strs) > 2 {
		return ErrLegacyInvalidDecimalStr
	}

	if lenDecs > LegacyPrecision {
		return fmt.Errorf("value '%s' exceeds max precision by %d decimal places: max precision %d", str, LegacyPrecision-lenDecs, LegacyPrecision)
	}

	// add some extra zero's to correct to the Precision factor
	zerosToAdd := LegacyPrecision - lenDecs
	zeros := strings.Repeat("0", zerosToAdd)
	combinedStr += zeros

	combined, ok := new(big.Int).SetString(combinedStr, 10) // base 10
	if !ok {
		return fmt.Errorf("failed to set decimal string with base 10: %s", combinedStr)
	}
	if combined.BitLen() > maxDecBitLen {
		return fmt.Errorf("decimal '%s' out of range; bitLen: got %d, max %d", str, combined.BitLen(), maxDecBitLen)
	}

	return nil
}

func ValidateMnemonic(mnemonic string) error {
	if !crypto.IsMnemonicValid(mnemonic) {
		return errors.New("invalid bip39 mnemonic")
	}
	return nil
}

// ValidateURLWithSchemes is a function to validate if a string is a valid URL and return an error if invalid.
// It allows for custom schemes to be specified.
func validateURLWithSchemes(str string, schemes ...string) error {
	u, err := url.Parse(str)
	if err != nil {
		return fmt.Errorf("invalid URL format: %v", err)
	}
	if u.Host == "" {
		return fmt.Errorf("URL is missing host")
	}
	if len(schemes) > 0 && !slices.Contains(schemes, u.Scheme) {
		return fmt.Errorf("URL must use one of the following schemes: %v", schemes)
	}
	return nil
}

// ValidateURL is a function to validate if a string is a valid URL and return an error if invalid.
// Only http and https schemes are allowed.
func ValidateURL(str string) error {
	return validateURLWithSchemes(str, "http", "https")
}

// ValidateURLWithPort is a function to validate if a string is a valid URL with a port number.
// Only http and https schemes are allowed, and the URL must include a valid port.
func ValidateURLWithPort(str string) error {
	u, err := url.Parse(str)
	if err != nil {
		return fmt.Errorf("invalid URL format: %v", err)
	}
	if u.Host == "" {
		return fmt.Errorf("URL is missing host")
	}
	if !slices.Contains([]string{"http", "https"}, u.Scheme) {
		return fmt.Errorf("URL must use one of the following schemes: [http https]")
	}

	// Check if port is present
	_, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		return fmt.Errorf("URL must include a port number")
	}

	// Validate port number
	portNum, err := strconv.Atoi(port)
	if err != nil || portNum < 1 || portNum > 65535 {
		return fmt.Errorf("invalid port number: must be between 1 and 65535")
	}

	return nil
}

// ValidateWSURL is a function to validate if a string is a valid WebSocket URL and return an error if invalid.
// Only ws and wss schemes are allowed.
func ValidateWSURL(str string) error {
	return validateURLWithSchemes(str, "ws", "wss")
}

// IsValidDNS checks if a given string is a valid DNS name
func IsValidDNS(dns string) bool {
	// Regular expression for validating DNS names
	dnsRegex := `^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`
	re := regexp.MustCompile(dnsRegex)

	// Validate DNS name
	return re.MatchString(dns)
}

// IsValidPeerOrSeed checks if each address in a comma-separated list is valid.
// It allows empty strings and returns an error with detailed reasons if any address is invalid.
// It accepts both IP addresses and DNS names.
func IsValidPeerOrSeed(addresses string) error {
	// Compile the regular expression for node ID
	nodeIDRegex, err := regexp.Compile(`^[a-f0-9]{40}$`)
	if err != nil {
		return fmt.Errorf("failed to compile nodeID regex: %v", err)
	}

	// Split the input string by commas to handle multiple addresses
	addressList := strings.Split(addresses, ",")

	var invalidAddresses []string

	// Iterate over each address and validate
	for _, address := range addressList {
		address = strings.TrimSpace(address) // Remove any leading/trailing spaces

		// Skip empty strings, as they're considered valid
		if address == "" {
			continue
		}

		parts := strings.Split(address, "@")
		if len(parts) != 2 {
			invalidAddresses = append(invalidAddresses, fmt.Sprintf("'%s': must be in format nodeID@ip_or_dns:port", address))
			continue
		}

		nodeID := parts[0]
		peerAddr := parts[1]

		// Validate node ID
		if !nodeIDRegex.MatchString(nodeID) {
			invalidAddresses = append(invalidAddresses, fmt.Sprintf("'%s': invalid node ID (must be 40-character hex string)", address))
			continue
		}

		// Split peer address into host (IP or DNS) and port
		host, port, err := net.SplitHostPort(peerAddr)
		if err != nil && !strings.Contains(err.Error(), "missing port in address") {
			invalidAddresses = append(invalidAddresses, fmt.Sprintf("'%s': invalid address (IP:Port or DNS:Port format required)", address))
			continue
		}

		// Validate host (can be IP or DNS)
		if net.ParseIP(host) == nil && !IsValidDNS(host) {
			invalidAddresses = append(invalidAddresses, fmt.Sprintf("'%s': invalid IP or DNS name", address))
			continue
		}

		// Validate port if present
		if port != "" {
			if _, err := fmt.Sscanf(port, "%d", new(int)); err != nil {
				invalidAddresses = append(invalidAddresses, fmt.Sprintf("'%s': invalid port", address))
				continue
			}
		}
	}

	// If there are any invalid addresses, return an error with detailed messages
	if len(invalidAddresses) > 0 {
		return errors.New("invalid peer/seed addresses: " + strings.Join(invalidAddresses, ", "))
	}

	return nil
}

func ValidateExactString(expect string) func(s string) error {
	return func(s string) error {
		if s != expect {
			return fmt.Errorf("please type `%s` to proceed", expect)
		}
		return nil
	}
}

func ValidateEmptyString(s string) error {
	if s == "" {
		return fmt.Errorf("cannot be empty string")
	}
	return nil
}

func IsValidInteger(s string) error {
	num, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("amount must be an integer")
	}
	if num < 0 {
		return fmt.Errorf("amount must be zero or a positive integer")
	}
	return nil
}

func IsValidAddress(s string) error {
	initBech32Regex := `^init1(?:[a-z0-9]{38}|[a-z0-9]{58})$`
	re := regexp.MustCompile(initBech32Regex)

	if !re.MatchString(s) {
		return errors.New("invalid address format")
	}
	return nil
}

func IsValidAddresses(s string) error {
	if s == "" {
		return nil
	}
	cache := make(map[string]bool)
	for _, address := range strings.Split(s, ",") {
		if err := IsValidAddress(address); err != nil {
			return err
		}
		if _, ok := cache[address]; ok {
			return fmt.Errorf("duplicate address: %s", address)
		}
		cache[address] = true
	}
	return nil
}

func ValidateNonEmptyAndLengthString(display string, maxLen int) func(s string) error {
	return func(s string) error {
		if s == "" {
			return fmt.Errorf("%s cannot be empty", display)
		}
		if len(s) > maxLen {
			return fmt.Errorf("%s is too long (max: %d)", display, maxLen)
		}
		return nil
	}
}

func IsValidTimestamp(s string) error {
	if _, err := time.ParseDuration(s); err != nil {
		return fmt.Errorf("invalid time format")
	}
	return nil
}

// LZ4 magic number for LZ4 frame format
var lz4MagicNumber = []byte{0x04, 0x22, 0x4D, 0x18}

// ValidateTarLz4Header checks if the downloaded file is a valid .tar.lz4 file based on the file header.
func ValidateTarLz4Header(dest string) error {
	// Open the .lz4 file
	file, err := os.Open(dest)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read the first few bytes of the file (header)
	header := make([]byte, 4)
	_, err = file.Read(header)
	if err != nil {
		return fmt.Errorf("failed to read file header: %w", err)
	}

	// Check if the header matches the LZ4 magic number
	if !bytes.Equal(header, lz4MagicNumber) {
		return fmt.Errorf("invalid file format: the file is not a valid .lz4 file")
	}

	// If the header matches, we assume it's a valid .lz4 file.
	// You could continue checking the contents further if needed, but this verifies the file is compressed with LZ4.

	return nil
}

func ValidatePositiveBigIntOrZero(s string) error {
	if s == "" {
		return errors.New("empty string is not a valid integer")
	}

	n := new(big.Int)
	_, ok := n.SetString(s, 10)
	if !ok {
		return fmt.Errorf("failed to parse '%s' as integer", s)
	}

	// Check if number is negative
	if n.Sign() < 0 {
		return fmt.Errorf("'%s' is a negative integer", s)
	}

	// Check if number is within uint256 range (2^256 - 1)
	maxUint256 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	if n.Cmp(maxUint256) > 0 {
		return fmt.Errorf("number exceeds maximum value (2^256 - 1)")
	}

	// For format validation
	expected := n.String()
	if s != expected {
		return fmt.Errorf("invalid integer format: '%s'", s)
	}

	return nil
}

func ValidatePositiveBigInt(s string) error {
	err := ValidatePositiveBigIntOrZero(s)
	if err != nil {
		return err
	}

	if s == "0" {
		return fmt.Errorf("'%s' is not a positive integer", s)
	}

	return nil
}

func ValidateUint64(s string) error {
	if s == "" {
		return errors.New("empty string is not a valid integer")
	}

	_, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse '%s' as uint64: %w", s, err)
	}

	return nil
}
