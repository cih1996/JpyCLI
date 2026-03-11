//go:build !windows

package cmd

import (
	"context"
	"os/exec"
)

func shellCmd(ctx context.Context, command string) *exec.Cmd {
	return exec.CommandContext(ctx, "sh", "-c", command)
}
