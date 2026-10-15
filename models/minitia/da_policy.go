package minitia

import (
	"fmt"
	"github.com/initia-labs/weave/common"
	"github.com/initia-labs/weave/types"
)

// ValidateLaunchDA validates DA configuration before creating a new rollup.
func ValidateLaunchDA(config *types.MinitiaConfig) error {
	if config == nil {
		return fmt.Errorf("rollup launch config is required")
	}
	if config.OpBridge == nil {
		return fmt.Errorf("rollup launch op_bridge config is required")
	}
	return common.ValidateNewDATarget(config.OpBridge.BatchSubmissionTarget)
}
