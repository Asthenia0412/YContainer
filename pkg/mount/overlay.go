package mount

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/Yancy/YContainer/internal/utils"
)

type OverlayOptions struct {
	LowerDir  string
	UpperDir  string
	WorkDir   string
	MergedDir string
}

type OverlayMount struct {
	opts   OverlayOptions
	logger *utils.Logger
}

func NewOverlayMount(opts OverlayOptions, logger *utils.Logger) *OverlayMount {
	return &OverlayMount{
		opts:   opts,
		logger: logger,
	}
}

func (o *OverlayMount) Mount() error {
	for _, dir := range []string{o.opts.UpperDir, o.opts.WorkDir, o.opts.MergedDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	optStr := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s",
		o.opts.LowerDir, o.opts.UpperDir, o.opts.WorkDir)
	flags := uintptr(syscall.MS_NODEV)

	if err := syscall.Mount("overlay", o.opts.MergedDir, "overlay", flags, optStr); err != nil {
		return fmt.Errorf("mount overlayfs: %w", err)
	}

	o.logger.Info("Mounted overlayfs: lower=%s upper=%s merged=%s",
		o.opts.LowerDir, o.opts.UpperDir, o.opts.MergedDir)
	return nil
}

func (o *OverlayMount) Unmount() error {
	if err := syscall.Unmount(o.opts.MergedDir, 0); err != nil {
		return fmt.Errorf("unmount overlayfs: %w", err)
	}
	o.logger.Info("Unmounted overlayfs: %s", o.opts.MergedDir)
	return nil
}

func SetupRootFS(rootfs string, logger *utils.Logger) error {
	if err := syscall.Mount("proc", filepath.Join(rootfs, "proc"), "proc", 0, ""); err != nil {
		return fmt.Errorf("mount proc: %w", err)
	}
	logger.Info("Mounted /proc")

	if err := syscall.Mount("sysfs", filepath.Join(rootfs, "sys"), "sysfs", 0, ""); err != nil {
		return fmt.Errorf("mount sysfs: %w", err)
	}
	logger.Info("Mounted /sys")

	if err := syscall.Mount("tmpfs", filepath.Join(rootfs, "tmp"), "tmpfs", 0, ""); err != nil {
		return fmt.Errorf("mount tmpfs: %w", err)
	}
	logger.Info("Mounted /tmp")

	if err := syscall.Mount("devtmpfs", filepath.Join(rootfs, "dev"), "devtmpfs", 0, ""); err != nil {
		return fmt.Errorf("mount devtmpfs: %w", err)
	}
	logger.Info("Mounted /dev")

	return nil
}

func PivotRoot(newRoot string) error {
	if err := syscall.Mount(newRoot, newRoot, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("bind mount new root: %w", err)
	}

	oldRoot := filepath.Join(newRoot, ".old_root")
	if err := os.MkdirAll(oldRoot, 0700); err != nil {
		return fmt.Errorf("create old root dir: %w", err)
	}

	if err := syscall.PivotRoot(newRoot, oldRoot); err != nil {
		return fmt.Errorf("pivot_root: %w", err)
	}

	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("chdir to /: %w", err)
	}

	oldRoot = "/.old_root"
	if err := syscall.Unmount(oldRoot, syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("unmount old root: %w", err)
	}
	if err := os.RemoveAll(oldRoot); err != nil {
		return fmt.Errorf("remove old root dir: %w", err)
	}

	return nil
}