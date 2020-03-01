package zeek

import (
	"context"
	"io"
	"os/exec"
	"path/filepath"
)

func Logify(ctx context.Context, dir string, pcap io.Reader) ([]string, error) {
	cmd := exec.CommandContext(ctx, "zeek", "-C", "-r", "-", "local")
	cmd.Stdin = pcap
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.log"))
	// Per filepath.Glob documentation the only possible error would be due to
	// an invalid glob pattern. Ok to panic.
	if err != nil {
		panic(err)
	}
	return files, nil
}
