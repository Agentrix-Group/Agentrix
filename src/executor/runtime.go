package executor

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/tracer"
)

var (
	DefaultEnvAllowlist = []string{
		"PATH=/usr/local/bin:/usr/bin:/bin",
		"PYTHONUNBUFFERED=1",
		"PYTHONDONTWRITEBYTECODE=1",
		"LANG=C.UTF-8",
		"LC_ALL=C.UTF-8",
	}
)

// BotRuntimeConfig specifies limits and options for spawning a bot process.
type BotRuntimeConfig struct {
	PlayerID      string
	CodePath      string
	Timeout       time.Duration
	MaxMemoryMB   int
	MaxOutputKB   int
	EnvAllowlist  []string
	AllowFallback bool
}

// BotProcess abstracts a running, isolated bot process with standard streams.
type BotProcess interface {
	Stdin() io.WriteCloser
	Stdout() io.Reader
	Stderr() io.Reader
	Kill() error
	Wait() error
	Pid() int
}

// BotRuntime is the isolation layer contract for running participant code.
type BotRuntime interface {
	Name() string
	IsAvailable() bool
	Spawn(ctx context.Context, cfg BotRuntimeConfig) (BotProcess, error)
}

type execBotProcess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
}

func (p *execBotProcess) Stdin() io.WriteCloser { return p.stdin }
func (p *execBotProcess) Stdout() io.Reader     { return p.stdout }
func (p *execBotProcess) Stderr() io.Reader     { return p.stderr }

func (p *execBotProcess) Kill() error {
	if p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Process.Kill()
	}
	return nil
}

func (p *execBotProcess) Wait() error {
	if p.cmd != nil {
		return p.cmd.Wait()
	}
	return nil
}

func (p *execBotProcess) Pid() int {
	if p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Process.Pid
	}
	return 0
}

func resolveBotPath(codePath string) (string, error) {
	resolved := codePath
	if _, err := os.Stat(resolved); err != nil {
		candidate := filepath.Join("../..", resolved)
		if _, candidateErr := os.Stat(candidate); candidateErr != nil {
			return "", fmt.Errorf("python bot not found: %s", codePath)
		}
		resolved = candidate
	}
	if !strings.EqualFold(filepath.Ext(resolved), ".py") {
		return "", fmt.Errorf("unsupported bot artifact %q: Agentrix MVP accepts only Python .py files", codePath)
	}
	absPath, err := filepath.Abs(resolved)
	if err != nil {
		return resolved, nil
	}
	return absPath, nil
}

// BubblewrapRuntime executes bots in a hardened rootless bubblewrap sandbox.
type BubblewrapRuntime struct{}

func (b *BubblewrapRuntime) Name() string { return "bubblewrap-rootless" }

func (b *BubblewrapRuntime) IsAvailable() bool {
	if os.Getenv("AGENTRIX_DISABLE_SANDBOX") == "1" {
		return false
	}
	_, err := exec.LookPath("bwrap")
	return err == nil
}

func (b *BubblewrapRuntime) Spawn(ctx context.Context, cfg BotRuntimeConfig) (BotProcess, error) {
	absPath, err := resolveBotPath(cfg.CodePath)
	if err != nil {
		return nil, err
	}

	// Minimal rootless sandbox mount list:
	// - --die-with-parent: Prevents orphaned child processes
	// - --unshare-net: Network completely unreachable
	// - --proc /proc: Private proc namespace
	// - --dev /dev: Minimal device nodes
	// - Only bind required system runtimes and the specific bot file
	args := []string{
		"--die-with-parent",
		"--unshare-net",
		"--tmpfs", "/",
		"--dev", "/dev",
		"--proc", "/proc",
		"--tmpfs", "/tmp",
	}

	for _, sysDir := range []string{"/usr", "/lib", "/lib64", "/bin", "/etc/alternatives", "/etc/ld.so.cache"} {
		if _, err := os.Stat(sysDir); err == nil {
			args = append(args, "--ro-bind", sysDir, sysDir)
		}
	}

	// Read-only mount of the specific bot file
	args = append(args, "--ro-bind", absPath, absPath)
	// Remount root read-only after setting up required mount points
	args = append(args, "--remount-ro", "/")
	args = append(args, "python3", absPath)

	cmd := exec.CommandContext(ctx, "bwrap", args...)

	// Clean environment: strictly pass only allowed runtime variables, stripping all host secrets
	env := DefaultEnvAllowlist
	if len(cfg.EnvAllowlist) > 0 {
		env = cfg.EnvAllowlist
	}
	cmd.Env = env

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start bubblewrap sandbox: %w", err)
	}
	tracer.RecordSandboxSpawn()

	return &execBotProcess{cmd: cmd, stdin: stdin, stdout: stdout, stderr: stderr}, nil
}

// PodmanRuntime executes bots using rootless Podman containers.
type PodmanRuntime struct {
	Image string
}

func (p *PodmanRuntime) Name() string { return "podman-rootless" }

func (p *PodmanRuntime) IsAvailable() bool {
	if os.Getenv("AGENTRIX_DISABLE_SANDBOX") == "1" {
		return false
	}
	_, err := exec.LookPath("podman")
	return err == nil
}

func (p *PodmanRuntime) Spawn(ctx context.Context, cfg BotRuntimeConfig) (BotProcess, error) {
	absPath, err := resolveBotPath(cfg.CodePath)
	if err != nil {
		return nil, err
	}

	image := p.Image
	if image == "" {
		image = "docker.io/library/python:3.11-slim"
	}

	args := []string{
		"run", "--rm", "-i",
		"--network", "none",
		"--read-only",
		"--tmpfs", "/tmp:rw,size=16m",
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges",
		"--memory", "256m",
		"--pids-limit", "64",
		"-v", fmt.Sprintf("%s:/bot/bot.py:ro", absPath),
		image,
		"python3", "/bot/bot.py",
	}

	cmd := exec.CommandContext(ctx, "podman", args...)
	cmd.Env = DefaultEnvAllowlist

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start podman container: %w", err)
	}
	tracer.RecordSandboxSpawn()

	return &execBotProcess{cmd: cmd, stdin: stdin, stdout: stdout, stderr: stderr}, nil
}

// DirectPythonRuntime is a development-only fallback runtime.
type DirectPythonRuntime struct{}

func (d *DirectPythonRuntime) Name() string { return "direct-python-dev" }

func (d *DirectPythonRuntime) IsAvailable() bool {
	_, err := exec.LookPath("python3")
	return err == nil
}

func (d *DirectPythonRuntime) Spawn(ctx context.Context, cfg BotRuntimeConfig) (BotProcess, error) {
	absPath, err := resolveBotPath(cfg.CodePath)
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, "python3", absPath)
	cmd.Env = DefaultEnvAllowlist

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start python3 process: %w", err)
	}
	tracer.RecordSandboxSpawn()

	return &execBotProcess{cmd: cmd, stdin: stdin, stdout: stdout, stderr: stderr}, nil
}

// FailClosedRuntime always rejects execution when no secure sandbox is available.
type FailClosedRuntime struct{}

func (f *FailClosedRuntime) Name() string { return "fail-closed" }

func (f *FailClosedRuntime) IsAvailable() bool { return true }

func (f *FailClosedRuntime) Spawn(ctx context.Context, cfg BotRuntimeConfig) (BotProcess, error) {
	return nil, ErrSandboxUnavailable
}

// DefaultBotRuntime resolves the appropriate bot runtime according to configuration.
func DefaultBotRuntime() BotRuntime {
	if os.Getenv("AGENTRIX_DISABLE_SANDBOX") == "1" {
		return &DirectPythonRuntime{}
	}

	bwrap := &BubblewrapRuntime{}
	if bwrap.IsAvailable() {
		return bwrap
	}

	podman := &PodmanRuntime{}
	if podman.IsAvailable() {
		return podman
	}

	// Production fail-closed
	return &FailClosedRuntime{}
}
