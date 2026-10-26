//go:build darwin || linux

package update

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
)

func lockBounded(path string) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create bounded update lock directory: %w", err)
	}
	file, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open bounded update lock: %w", err)
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("bounded update already running: %w", err)
	}
	return func() { _ = file.Close() }, nil
}

func checkResources(options dto.BoundedUpdateOptions) error {
	if options.MinFreeGiB > 0 {
		var stat syscall.Statfs_t
		if err := syscall.Statfs("/", &stat); err != nil {
			return fmt.Errorf("measure free disk: %w", err)
		}
		free := stat.Bavail * uint64(stat.Bsize) / (1024 * 1024 * 1024)
		if free < options.MinFreeGiB {
			return fmt.Errorf("free disk %d GiB is below %d GiB", free, options.MinFreeGiB)
		}
	}
	if options.MaxSwapGiB > 0 && runtime.GOOS == "darwin" {
		if err := checkDarwinMemoryPressure(); err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		out, err := exec.CommandContext(ctx, "/usr/sbin/sysctl", "-n", "vm.swapusage").Output()
		if err != nil {
			return fmt.Errorf("measure macOS swap: %w", err)
		}
		return parseDarwinSwap(string(out), options.MaxSwapGiB)
	}
	return nil
}

func checkDarwinMemoryPressure() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "/usr/bin/memory_pressure", "-Q").CombinedOutput()
	if err != nil {
		return fmt.Errorf("measure macOS memory pressure: %w", err)
	}
	return parseDarwinPressure(string(out))
}

func parseDarwinSwap(output string, maximum uint64) error {
	match := regexp.MustCompile(`used = ([0-9.]+)M`).FindStringSubmatch(output)
	if len(match) != 2 {
		return fmt.Errorf("measure macOS swap: unexpected output")
	}
	used, err := strconv.ParseFloat(match[1], 64)
	if err != nil || used >= float64(maximum)*1024 {
		return fmt.Errorf("macOS swap is at or above %d GiB", maximum)
	}
	return nil
}

func parseDarwinPressure(output string) error {
	match := regexp.MustCompile(`System-wide memory free percentage: (\d+)%`).FindStringSubmatch(output)
	if len(match) != 2 {
		return fmt.Errorf("measure macOS memory pressure: unexpected output")
	}
	free, err := strconv.Atoi(match[1])
	if err != nil || free < 10 || free > 100 {
		return fmt.Errorf("macOS memory pressure is too high")
	}
	return nil
}
