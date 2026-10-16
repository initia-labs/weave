package common

import (
	"errors"
	"fmt"

	"github.com/initia-labs/weave/internal/features"
)

// ErrCelestiaDADisabled identifies new Celestia DA operations forbidden by this build.
var ErrCelestiaDADisabled = errors.New("Celestia DA is disabled in this build; use Initia DA or a Celestia-enabled build")

// ValidateNewDATarget validates a DA target for a new rollup operation.
func ValidateNewDATarget(target string) error {
	switch target {
	case "INITIA":
		return nil
	case "CELESTIA":
		if !features.CelestiaDAEnabled {
			return fmt.Errorf("DA target %q: %w", target, ErrCelestiaDADisabled)
		}
		return nil
	default:
		return fmt.Errorf("unsupported DA target %q: expected INITIA or CELESTIA", target)
	}
}

// ValidateNewDAPrefix validates a DA address prefix for a new executor operation.
func ValidateNewDAPrefix(prefix string) error {
	switch prefix {
	case "init":
		return nil
	case "celestia":
		if !features.CelestiaDAEnabled {
			return fmt.Errorf("DA prefix %q: %w", prefix, ErrCelestiaDADisabled)
		}
		return nil
	default:
		return fmt.Errorf("unsupported DA prefix %q: expected init or celestia", prefix)
	}
}
