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

	"github.com/Agentrix-Group/Agentrix/src/model"
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

// botMountPoint es donde el bot ve su código dentro del sandbox.
const botMountPoint = "/bot"

// botMount describe qué se expone al bot en /bot (ADR-0014): la carpeta
// completa de un paquete v2, o solo el archivo de un bot v1 (que no debe
// ver el resto de su carpeta).
type botMount struct {
	hostPath string // carpeta del paquete (v2) o archivo del bot (v1)
	isBundle bool
	workDir  string // directorio de trabajo en el host (runtime directo)
}

func resolveBotMount(codePath string) (botMount, error) {
	absPath, err := resolveBotPath(codePath)
	if err != nil {
		return botMount{}, err
	}
	if dir, ok := model.BotBundleDir(absPath); ok && filepath.Base(absPath) == "bot.py" {
		return botMount{hostPath: dir, isBundle: true, workDir: dir}, nil
	}
	return botMount{hostPath: absPath, workDir: filepath.Dir(absPath)}, nil
}

// bubblewrapArgs arma el sandbox: raíz vacía en solo lectura, sin red, los
// runtimes del sistema en solo lectura y el código del bot en /bot, también
// en solo lectura, con /bot como directorio de trabajo.
func bubblewrapArgs(mount botMount, systemDirs []string) []string {
	args := []string{
		"--die-with-parent",
		"--unshare-net",
		"--tmpfs", "/",
		"--dev", "/dev",
		"--proc", "/proc",
		"--tmpfs", "/tmp",
	}
	for _, sysDir := range systemDirs {
		args = append(args, "--ro-bind", sysDir, sysDir)
	}
	if mount.isBundle {
		args = append(args, "--ro-bind", mount.hostPath, botMountPoint)
	} else {
		args = append(args, "--ro-bind", mount.hostPath, botMountPoint+"/bot.py")
	}
	// Remount root read-only after setting up required mount points
	args = append(args, "--remount-ro", "/", "--chdir", botMountPoint)
	return append(args, "python3", botMountPoint+"/bot.py")
}

// podmanArgs arma el contenedor con el código del bot en /bot, en solo
// lectura, y /bot como directorio de trabajo.
func podmanArgs(mount botMount, image string) []string {
	volume := fmt.Sprintf("%s:%s/bot.py:ro", mount.hostPath, botMountPoint)
	if mount.isBundle {
		volume = fmt.Sprintf("%s:%s:ro", mount.hostPath, botMountPoint)
	}
	return []string{
		"run", "--rm", "-i",
		"--network", "none",
		"--read-only",
		"--tmpfs", "/tmp:rw,size=16m",
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges",
		"--memory", "256m",
		"--pids-limit", "64",
		"-v", volume,
		"-w", botMountPoint,
		image,
		"python3", botMountPoint + "/bot.py",
	}
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
	mount, err := resolveBotMount(cfg.CodePath)
	if err != nil {
		return nil, err
	}
	var systemDirs []string
	for _, sysDir := range []string{"/usr", "/lib", "/lib64", "/bin", "/etc/alternatives", "/etc/ld.so.cache"} {
		if _, err := os.Stat(sysDir); err == nil {
			systemDirs = append(systemDirs, sysDir)
		}
	}
	args := bubblewrapArgs(mount, systemDirs)

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
	mount, err := resolveBotMount(cfg.CodePath)
	if err != nil {
		return nil, err
	}
	image := p.Image
	if image == "" {
		image = "docker.io/library/python:3.11-slim"
	}
	args := podmanArgs(mount, image)

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
	mount, err := resolveBotMount(cfg.CodePath)
	if err != nil {
		return nil, err
	}
	entry := mount.hostPath
	if mount.isBundle {
		entry = filepath.Join(mount.hostPath, "bot.py")
	}
	cmd := exec.CommandContext(ctx, "python3", entry)
	// Sin aislamiento (solo desarrollo): al menos el mismo directorio de
	// trabajo que ve el bot en el sandbox.
	cmd.Dir = mount.workDir
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
