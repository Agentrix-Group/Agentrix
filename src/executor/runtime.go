// Package executor runs matches: it owns the bot sandbox, the bot protocol
// session, the engine subprocess loop and the worker that reserves jobs.
package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/model"
)

// BotSpec is everything a runtime needs to start one bot process.
type BotSpec struct {
	CodePath string // host path of the verified entrypoint file
	Runtime  string // e.g. "python3"
	Limits   model.ExecutionLimits
}

// BotProcess is a running, isolated bot.
type BotProcess interface {
	Stdin() io.WriteCloser
	Stdout() io.Reader
	// Stderr returns the bounded tail of the bot's stderr.
	Stderr() string
	Kill() error
	Wait() error
}

// BotRuntime isolates untrusted bot code.
type BotRuntime interface {
	Name() string
	// Check verifies that the runtime can enforce its guarantees on this
	// host. Workers refuse to start when it fails (fail closed).
	Check() error
	Spawn(ctx context.Context, spec BotSpec) (BotProcess, error)
}

// ErrSpawn wraps failures to start a sandboxed process: they are
// infrastructure failures, never a bot verdict.
var ErrSpawn = errors.New("sandbox spawn failed")

// NewRuntime returns the configured runtime. There is no silent fallback: a
// missing sandbox is reported by Check.
func NewRuntime(name, podmanImage string) (BotRuntime, error) {
	switch name {
	case "bubblewrap":
		return &BubblewrapRuntime{}, nil
	case "podman":
		return &PodmanRuntime{Image: podmanImage}, nil
	case "direct":
		return &DirectRuntime{}, nil
	}
	return nil, fmt.Errorf("unknown sandbox runtime %q", name)
}

var sandboxEnv = []string{"PATH=/usr/local/bin:/usr/bin:/bin", "PYTHONUNBUFFERED=1", "PYTHONDONTWRITEBYTECODE=1",
	"PYTHONHASHSEED=0", "LANG=C.UTF-8", "LC_ALL=C.UTF-8", "HOME=/tmp"}

func interpreter(runtime string) ([]string, error) {
	switch runtime {
	case "python3":
		return []string{"python3", "-I", "-B", "-u"}, nil
	}
	return nil, fmt.Errorf("unsupported bot runtime %q", runtime)
}

// prlimitArgs are applied inside the sandbox namespace (measured: RLIMIT_NPROC
// applied inside a user namespace counts only the sandbox's processes).
func prlimitArgs(l model.ExecutionLimits) []string {
	return []string{"prlimit",
		"--as=" + strconv.FormatInt(int64(l.MemoryMB)<<20, 10),
		"--cpu=" + strconv.Itoa(l.CPUSeconds),
		"--nproc=" + strconv.Itoa(l.MaxProcesses),
		"--fsize=" + strconv.FormatInt(int64(l.MaxFileSizeKB)<<10, 10),
		"--nofile=64",
		"--core=0",
		"--"}
}

// BubblewrapRuntime: unprivileged namespaces (user, pid, net, ipc, uts,
// cgroup), read-only minimal filesystem, bounded tmpfs, no network, clean
// environment, and rlimits for memory, CPU time, processes and file size.
type BubblewrapRuntime struct{}

func (b *BubblewrapRuntime) Name() string { return "bubblewrap" }

func (b *BubblewrapRuntime) Check() error {
	for _, bin := range []string{"bwrap", "prlimit", "python3"} {
		if _, err := exec.LookPath(bin); err != nil {
			return fmt.Errorf("bubblewrap runtime requires %s: %w", bin, err)
		}
	}
	probeFile, err := os.CreateTemp("", "agentrix-sandbox-probe-*.py")
	if err != nil {
		return err
	}
	defer os.Remove(probeFile.Name())
	if _, err := probeFile.WriteString("pass\n"); err != nil {
		return err
	}
	if err := probeFile.Close(); err != nil {
		return err
	}
	args, err := b.args(BotSpec{CodePath: probeFile.Name(), Runtime: "python3", Limits: model.ExecutionLimits{
		MemoryMB: 256, CPUSeconds: 5, MaxProcesses: 8, MaxFileSizeKB: 64, MaxStderrKB: 4}})
	if err != nil {
		return err
	}
	// Run the exact sandbox command line with an empty program: namespaces,
	// mounts, rlimits and the interpreter must all work.
	probe := exec.Command("bwrap", args...)
	if out, err := probe.CombinedOutput(); err != nil {
		return fmt.Errorf("bubblewrap cannot create namespaces: %v (%s)", err, bytes.TrimSpace(out))
	}
	return nil
}

func (b *BubblewrapRuntime) args(spec BotSpec) ([]string, error) {
	interp, err := interpreter(spec.Runtime)
	if err != nil {
		return nil, err
	}
	args := []string{
		"--die-with-parent", "--new-session", "--unshare-all", "--cap-drop", "ALL", "--clearenv",
		"--ro-bind", "/usr", "/usr",
		"--proc", "/proc", "--dev", "/dev",
		"--size", strconv.FormatInt(int64(spec.Limits.MaxFileSizeKB)<<10, 10), "--tmpfs", "/tmp",
		"--ro-bind", spec.CodePath, "/bot/bot.py",
		"--chdir", "/bot",
	}
	for _, link := range [][2]string{{"usr/bin", "/bin"}, {"usr/lib", "/lib"}, {"usr/lib64", "/lib64"}, {"usr/sbin", "/sbin"}} {
		if info, err := os.Lstat(link[1]); err == nil && info.Mode()&os.ModeSymlink != 0 {
			args = append(args, "--symlink", link[0], link[1])
		} else if err == nil && info.IsDir() {
			args = append(args, "--ro-bind", link[1], link[1])
		}
	}
	for _, f := range []string{"/etc/ld.so.cache", "/etc/localtime"} {
		if _, err := os.Stat(f); err == nil {
			args = append(args, "--ro-bind", f, f)
		}
	}
	for _, kv := range sandboxEnv {
		name, value, _ := cutEnv(kv)
		args = append(args, "--setenv", name, value)
	}
	args = append(args, prlimitArgs(spec.Limits)...)
	args = append(args, interp...)
	return append(args, "/bot/bot.py"), nil
}

func (b *BubblewrapRuntime) Spawn(ctx context.Context, spec BotSpec) (BotProcess, error) {
	args, err := b.args(spec)
	if err != nil {
		return nil, err
	}
	return startProcess(ctx, exec.Command("bwrap", args...), spec.Limits)
}

// PodmanRuntime runs each bot in a rootless container with cgroup limits.
type PodmanRuntime struct{ Image string }

func (p *PodmanRuntime) Name() string { return "podman" }

func (p *PodmanRuntime) Check() error {
	if _, err := exec.LookPath("podman"); err != nil {
		return fmt.Errorf("podman runtime requires podman: %w", err)
	}
	if out, err := exec.Command("podman", "image", "exists", p.Image).CombinedOutput(); err != nil {
		return fmt.Errorf("podman image %s is not available locally: %v (%s)", p.Image, err, bytes.TrimSpace(out))
	}
	return nil
}

func (p *PodmanRuntime) Spawn(ctx context.Context, spec BotSpec) (BotProcess, error) {
	interp, err := interpreter(spec.Runtime)
	if err != nil {
		return nil, err
	}
	l := spec.Limits
	args := []string{"run", "--rm", "-i", "--network", "none", "--read-only", "--cap-drop", "ALL",
		"--security-opt", "no-new-privileges", "--userns", "keep-id",
		"--memory", strconv.Itoa(l.MemoryMB) + "m", "--memory-swap", strconv.Itoa(l.MemoryMB) + "m",
		"--pids-limit", strconv.Itoa(l.MaxProcesses), "--cpus", "1",
		"--ulimit", "cpu=" + strconv.Itoa(l.CPUSeconds), "--ulimit", "fsize=" + strconv.FormatInt(int64(l.MaxFileSizeKB)<<10, 10),
		"--tmpfs", "/tmp:rw,size=" + strconv.Itoa(l.MaxFileSizeKB) + "k", "--workdir", "/bot",
		"-v", spec.CodePath + ":/bot/bot.py:ro,Z", "--env-host=false"}
	for _, kv := range sandboxEnv {
		args = append(args, "--env", kv)
	}
	args = append(args, p.Image)
	args = append(args, interp...)
	args = append(args, "/bot/bot.py")
	return startProcess(ctx, exec.Command("podman", args...), spec.Limits)
}

// DirectRuntime runs python3 without isolation. Configuration only allows
// it in dev/test (config.Load rejects it elsewhere).
type DirectRuntime struct{}

func (d *DirectRuntime) Name() string { return "direct-unsandboxed" }

func (d *DirectRuntime) Check() error {
	_, err := exec.LookPath("python3")
	return err
}

func (d *DirectRuntime) Spawn(ctx context.Context, spec BotSpec) (BotProcess, error) {
	interp, err := interpreter(spec.Runtime)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(interp[0], append(interp[1:], spec.CodePath)...)
	cmd.Dir = filepath.Dir(spec.CodePath)
	return startProcess(ctx, cmd, spec.Limits)
}

func cutEnv(kv string) (string, string, bool) {
	for i := 0; i < len(kv); i++ {
		if kv[i] == '=' {
			return kv[:i], kv[i+1:], true
		}
	}
	return kv, "", false
}

// ---------------------------------------------------------------------------
// Process plumbing
// ---------------------------------------------------------------------------

type execProcess struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.Reader
	stderr  *tailBuffer
	once    sync.Once
	exited  chan struct{}
	waitErr error
}

func startProcess(ctx context.Context, cmd *exec.Cmd, limits model.ExecutionLimits) (BotProcess, error) {
	if cmd.Env == nil {
		cmd.Env = append([]string(nil), sandboxEnv...)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pdeathsig: syscall.SIGKILL}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSpawn, err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSpawn, err)
	}
	tail := &tailBuffer{limit: limits.MaxStderrKB << 10}
	cmd.Stderr = tail
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSpawn, err)
	}
	p := &execProcess{cmd: cmd, stdin: stdin, stdout: stdout, stderr: tail, exited: make(chan struct{})}
	go func() {
		p.waitErr = cmd.Wait()
		close(p.exited)
	}()
	go func() {
		<-ctx.Done()
		_ = p.Kill()
	}()
	return p, nil
}

func (p *execProcess) Stdin() io.WriteCloser { return p.stdin }
func (p *execProcess) Stdout() io.Reader     { return p.stdout }
func (p *execProcess) Stderr() string        { return p.stderr.String() }

// Kill terminates the whole process group of the sandbox.
func (p *execProcess) Kill() error {
	var err error
	p.once.Do(func() {
		if p.cmd.Process != nil {
			err = syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL)
			if errors.Is(err, syscall.ESRCH) {
				err = nil
			}
		}
	})
	return err
}

func (p *execProcess) Wait() error {
	<-p.exited
	return p.waitErr
}

// ResourceKilled reports whether the process ended by a resource limit
// signal (SIGXCPU from RLIMIT_CPU, or SIGKILL from the hard limit). bwrap
// and prlimit propagate a signal death as exit code 128+signal.
func (p *execProcess) ResourceKilled(wait time.Duration) bool {
	select {
	case <-p.exited:
	case <-time.After(wait):
		return false
	}
	state := p.cmd.ProcessState
	if state == nil {
		return false
	}
	if ws, ok := state.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		return ws.Signal() == syscall.SIGXCPU
	}
	code := state.ExitCode()
	return code == 128+int(syscall.SIGXCPU) || code == 128+int(syscall.SIGKILL)
}

// tailBuffer keeps the last limit bytes written to it.
type tailBuffer struct {
	mu    sync.Mutex
	limit int
	buf   []byte
	total int64
}

func (t *tailBuffer) Write(b []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.total += int64(len(b))
	t.buf = append(t.buf, b...)
	if over := len(t.buf) - t.limit; over > 0 {
		t.buf = append([]byte(nil), t.buf[over:]...)
	}
	return len(b), nil
}

func (t *tailBuffer) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return string(bytes.ToValidUTF8(t.buf, nil))
}
