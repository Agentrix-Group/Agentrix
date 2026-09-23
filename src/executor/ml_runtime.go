package executor

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"golang.org/x/sys/unix"
)

// Límites del runtime python-ml-cpu (ADR-0014).
const (
	// mlMemoryLimitBytes limita el espacio de direcciones del bot. Medido:
	// un bot ONNX de ejemplo reserva ~450 MB virtuales y usa ~65 MB reales.
	mlMemoryLimitBytes = 1 << 30
	// mlInitTimeout es el tiempo para la primera respuesta, que incluye
	// cargar el modelo.
	mlInitTimeout = 10 * time.Second
)

// ErrRuntimeUnavailable indica que el runtime pedido no está instalado en
// este worker: la ejecución falla cerrada.
var ErrRuntimeUnavailable = errors.New("bot runtime unavailable")

// mlThreadEnv fuerza un hilo en las librerías numéricas: cada bot ML usa
// un núcleo, por paridad entre bots y reproducibilidad.
var mlThreadEnv = []string{
	"OMP_NUM_THREADS=1",
	"OPENBLAS_NUM_THREADS=1",
	"MKL_NUM_THREADS=1",
	"NUMEXPR_NUM_THREADS=1",
	"VECLIB_MAXIMUM_THREADS=1",
}

// pythonRuntime describe cómo ejecutar el Python de un bot.
type pythonRuntime struct {
	name        string
	interpreter string   // "python3" del sistema o el del runtime ML
	bindDir     string   // carpeta de runtimes a montar en solo lectura
	env         []string // entorno completo del proceso
	wrapper     []string // prlimit/taskset antes del sandbox
	initTimeout time.Duration
}

// BotRuntimesDir es la carpeta con los runtimes construidos por
// `make bot-runtimes` (AGENTRIX_BOT_RUNTIMES_DIR o bin/runtimes).
func BotRuntimesDir() string {
	if dir := os.Getenv("AGENTRIX_BOT_RUNTIMES_DIR"); dir != "" {
		return dir
	}
	for _, candidate := range []string{"bin/runtimes", "../../bin/runtimes"} {
		if abs, err := filepath.Abs(candidate); err == nil {
			if _, err := os.Stat(abs); err == nil {
				return abs
			}
		}
	}
	return "bin/runtimes"
}

// resolvePythonRuntime devuelve la configuración de ejecución del runtime
// `name`. cpu es el núcleo asignado al bot (solo python-ml-cpu).
func resolvePythonRuntime(name string, cpu int) (pythonRuntime, error) {
	switch name {
	case "", model.BotRuntimePythonStdlib:
		return pythonRuntime{name: model.BotRuntimePythonStdlib, interpreter: "python3", env: DefaultEnvAllowlist}, nil
	case model.BotRuntimePythonMLCPU:
		dir, err := filepath.Abs(BotRuntimesDir())
		if err != nil {
			return pythonRuntime{}, err
		}
		interpreter := filepath.Join(dir, "python-ml-cpu", "bin", "python")
		if _, err := os.Stat(interpreter); err != nil {
			return pythonRuntime{}, fmt.Errorf("%w: %s is not installed on this worker (run make bot-runtimes)", ErrRuntimeUnavailable, name)
		}
		env := append(append([]string{}, DefaultEnvAllowlist...), mlThreadEnv...)
		limit := strconv.Itoa(mlMemoryLimitBytes)
		return pythonRuntime{
			name:        name,
			interpreter: interpreter,
			bindDir:     dir,
			env:         env,
			wrapper:     []string{"prlimit", "--as=" + limit + ":" + limit, "taskset", "-c", strconv.Itoa(cpu)},
			initTimeout: mlInitTimeout,
		}, nil
	default:
		return pythonRuntime{}, fmt.Errorf("%w: unknown runtime %q", ErrRuntimeUnavailable, name)
	}
}

// allowedCPUs devuelve los núcleos que este proceso tiene permitidos (en un
// contenedor pueden no ser 0..n-1).
func allowedCPUs() []int {
	var set unix.CPUSet
	if err := unix.SchedGetaffinity(0, &set); err != nil {
		return []int{0}
	}
	var cpus []int
	for cpu := 0; cpu < 1024 && len(cpus) < set.Count(); cpu++ {
		if set.IsSet(cpu) {
			cpus = append(cpus, cpu)
		}
	}
	if len(cpus) == 0 {
		return []int{0}
	}
	return cpus
}
