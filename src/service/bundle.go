package service

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/Agentrix-Group/Agentrix/src/model"
)

// Límites del paquete de bot v2 (ADR-0014).
const (
	// MaxBundleBytes es el tamaño máximo del ZIP subido y también del
	// contenido descomprimido total.
	MaxBundleBytes = 50 << 20
	maxModuleBytes = 1 << 20
	maxModelBytes  = 20 << 20
	maxBundleFiles = 64
	// Contenido descomprimido máximo dentro de un .npz (también es un ZIP).
	maxNpzUncompressed = 64 << 20
)

// Runtimes de ejecución admitidos en agentrix.json.
const (
	RuntimePythonStdlib = "python-stdlib"
	RuntimePythonMLCPU  = "python-ml-cpu"
)

var (
	moduleName      = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*\.py$`)
	modelSegment    = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]*$`)
	modelExtensions = map[string]bool{".onnx": true, ".safetensors": true, ".npz": true, ".json": true}
)

// BotBundle es un paquete de bot ya validado.
type BotBundle struct {
	Manifest      model.AgentPackageManifest
	ManifestBytes []byte
	// Files contiene todos los archivos del paquete, incluido agentrix.json,
	// con su ruta relativa normalizada.
	Files map[string][]byte
	// FileDigests es el SHA-256 hexadecimal de cada archivo.
	FileDigests map[string]string
	// Digest identifica el paquete completo: SHA-256 del listado ordenado
	// "ruta\x00sha256\n" de todos sus archivos.
	Digest string
}

// SortedPaths devuelve las rutas del paquete en orden estable.
func (b *BotBundle) SortedPaths() []string {
	paths := make([]string, 0, len(b.Files))
	for name := range b.Files {
		paths = append(paths, name)
	}
	sort.Strings(paths)
	return paths
}

func bundleError(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidBotBundle, fmt.Sprintf(format, args...))
}

// classifyBundlePath valida una ruta del ZIP y devuelve su límite de tamaño.
// Se admiten agentrix.json y módulos .py en la raíz, y archivos de modelo
// (.onnx, .safetensors, .npz, .json) dentro de model/.
func classifyBundlePath(name string) (limit uint64, isModel bool, err error) {
	if name == "agentrix.json" {
		return maxModuleBytes, false, nil
	}
	if !strings.Contains(name, "/") {
		if moduleName.MatchString(name) {
			return maxModuleBytes, false, nil
		}
		return 0, false, bundleError("unexpected file %s (root accepts agentrix.json and .py modules)", name)
	}
	segments := strings.Split(name, "/")
	if segments[0] != "model" || len(segments) < 2 {
		return 0, false, bundleError("unexpected file %s (model files go under model/)", name)
	}
	for _, segment := range segments[1:] {
		if !modelSegment.MatchString(segment) {
			return 0, false, bundleError("invalid path segment in %s", name)
		}
	}
	if !modelExtensions[strings.ToLower(path.Ext(name))] {
		return 0, false, bundleError("model file %s must be .onnx, .safetensors, .npz or .json", name)
	}
	return maxModelBytes, true, nil
}

func readBotBundle(archive []byte) (*BotBundle, error) {
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, bundleError("malformed ZIP")
	}
	files := make(map[string][]byte)
	var total uint64
	for _, item := range reader.File {
		if item.FileInfo().IsDir() {
			continue
		}
		name := path.Clean(strings.ReplaceAll(item.Name, "\\", "/"))
		if name != item.Name || name == "." || strings.HasPrefix(name, "/") || strings.HasPrefix(name, "../") || name == ".." {
			return nil, bundleError("unsafe archive path %q", item.Name)
		}
		if !item.Mode().IsRegular() {
			return nil, bundleError("%s is not a regular file", name)
		}
		limit, _, err := classifyBundlePath(name)
		if err != nil {
			return nil, err
		}
		if _, duplicated := files[name]; duplicated {
			return nil, bundleError("duplicate file %s", name)
		}
		if len(files) >= maxBundleFiles {
			return nil, bundleError("more than %d files", maxBundleFiles)
		}
		if item.UncompressedSize64 > limit {
			return nil, bundleError("file %s exceeds %d bytes", name, limit)
		}
		total += item.UncompressedSize64
		if total > MaxBundleBytes {
			return nil, bundleError("uncompressed content exceeds %d bytes", MaxBundleBytes)
		}
		stream, err := item.Open()
		if err != nil {
			return nil, bundleError("cannot open %s", name)
		}
		// Protección contra ZIP bombs: el tamaño declarado se revisa antes de
		// descomprimir y, como puede mentir, la lectura se corta en el límite
		// + 1 byte. La memoria usada nunca supera los límites del paquete,
		// sin importar la razón de compresión.
		content, readErr := io.ReadAll(io.LimitReader(stream, int64(limit)+1))
		closeErr := stream.Close()
		if readErr != nil || closeErr != nil {
			return nil, bundleError("cannot read %s", name)
		}
		if uint64(len(content)) > limit || uint64(len(content)) != item.UncompressedSize64 {
			return nil, bundleError("file %s does not match its declared size", name)
		}
		files[name] = content
	}

	manifestBytes, manifestOK := files["agentrix.json"]
	botBytes, botOK := files["bot.py"]
	if !manifestOK || !botOK || len(botBytes) == 0 {
		return nil, bundleError("agentrix.json and bot.py are required")
	}
	var manifest model.AgentPackageManifest
	decoder := json.NewDecoder(bytes.NewReader(manifestBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return nil, bundleError("invalid agentrix.json")
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return nil, bundleError("invalid agentrix.json")
	}
	if manifest.Name == "" || len(manifest.Name) > 80 {
		return nil, bundleError("manifest name must contain 1 to 80 characters")
	}
	switch manifest.Runtime {
	case "":
		manifest.Runtime = RuntimePythonStdlib
	case RuntimePythonStdlib, RuntimePythonMLCPU:
	default:
		return nil, bundleError("runtime must be %s or %s", RuntimePythonStdlib, RuntimePythonMLCPU)
	}

	digests := make(map[string]string, len(files))
	for name, content := range files {
		if _, isModel, _ := classifyBundlePath(name); isModel {
			if err := validateModelFile(name, content); err != nil {
				return nil, err
			}
		}
		sum := sha256.Sum256(content)
		digests[name] = hex.EncodeToString(sum[:])
	}
	bundle := &BotBundle{Manifest: manifest, ManifestBytes: manifestBytes, Files: files, FileDigests: digests}
	bundle.Digest = model.BundleListingDigest(digests)
	return bundle, nil
}

// validateModelFile aplica la validación estática de ADR-0014 según la
// extensión. Los .onnx solo se validan por tamaño acá; su carga real se
// prueba en la admisión con el runtime del bot.
func validateModelFile(name string, content []byte) error {
	if len(content) == 0 {
		return bundleError("model file %s is empty", name)
	}
	switch strings.ToLower(path.Ext(name)) {
	case ".json":
		if !json.Valid(content) {
			return bundleError("model file %s is not valid JSON", name)
		}
	case ".npz":
		if err := validateNpz(content); err != nil {
			return bundleError("model file %s: %v", name, err)
		}
	case ".safetensors":
		if err := validateSafetensors(content); err != nil {
			return bundleError("model file %s: %v", name, err)
		}
	}
	return nil
}

// npyKinds son los tipos de dato numéricos admitidos en .npy (booleano,
// enteros, flotantes y complejos). Se excluyen objetos de Python ('O', que
// numpy solo carga con pickle), void/estructurados, strings y fechas.
var npyKinds = map[byte]bool{'b': true, 'i': true, 'u': true, 'f': true, 'c': true, '?': true}

var npyDescr = regexp.MustCompile(`'descr'\s*:\s*'([<>|=])([a-zA-Z?])(\d*)'`)

func validateNpz(content []byte) error {
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return errors.New("not a valid .npz archive")
	}
	if len(reader.File) == 0 {
		return errors.New("empty .npz archive")
	}
	var total uint64
	for _, item := range reader.File {
		if !strings.HasSuffix(item.Name, ".npy") || strings.Contains(item.Name, "/") || strings.Contains(item.Name, "..") {
			return fmt.Errorf("unexpected entry %q", item.Name)
		}
		total += item.UncompressedSize64
		if total > maxNpzUncompressed {
			return errors.New("arrays exceed the uncompressed size limit")
		}
		stream, err := item.Open()
		if err != nil {
			return fmt.Errorf("cannot open %s", item.Name)
		}
		header, err := readNpyHeader(stream)
		stream.Close()
		if err != nil {
			return fmt.Errorf("%s: %v", item.Name, err)
		}
		match := npyDescr.FindStringSubmatch(header)
		if match == nil {
			return fmt.Errorf("%s: only plain numeric arrays are allowed", item.Name)
		}
		if !npyKinds[match[2][0]] {
			return fmt.Errorf("%s: dtype %q is not allowed (no Python objects)", item.Name, match[1]+match[2]+match[3])
		}
	}
	return nil
}

func readNpyHeader(stream io.Reader) (string, error) {
	prefix := make([]byte, 10)
	if _, err := io.ReadFull(stream, prefix); err != nil {
		return "", errors.New("truncated header")
	}
	if !bytes.Equal(prefix[:6], []byte("\x93NUMPY")) {
		return "", errors.New("not a .npy array")
	}
	var headerLen uint32
	switch prefix[6] {
	case 1:
		headerLen = uint32(binary.LittleEndian.Uint16(prefix[8:10]))
	case 2, 3:
		rest := make([]byte, 2)
		if _, err := io.ReadFull(stream, rest); err != nil {
			return "", errors.New("truncated header")
		}
		headerLen = binary.LittleEndian.Uint32(append(prefix[8:10:10], rest...))
	default:
		return "", fmt.Errorf("unsupported .npy version %d", prefix[6])
	}
	if headerLen > 1<<16 {
		return "", errors.New("header too large")
	}
	header := make([]byte, headerLen)
	if _, err := io.ReadFull(stream, header); err != nil {
		return "", errors.New("truncated header")
	}
	return string(header), nil
}

// safetensorsDtypes son los tipos de safetensors admitidos, con su tamaño
// en bytes por elemento.
var safetensorsDtypes = map[string]uint64{
	"BOOL": 1, "U8": 1, "I8": 1, "F8_E4M3": 1, "F8_E5M2": 1,
	"U16": 2, "I16": 2, "F16": 2, "BF16": 2,
	"U32": 4, "I32": 4, "F32": 4,
	"U64": 8, "I64": 8, "F64": 8,
}

func validateSafetensors(content []byte) error {
	if len(content) < 8 {
		return errors.New("truncated header")
	}
	headerLen := binary.LittleEndian.Uint64(content[:8])
	if headerLen == 0 || headerLen > 100<<20 || 8+headerLen > uint64(len(content)) {
		return errors.New("invalid header length")
	}
	var header map[string]json.RawMessage
	if err := json.Unmarshal(content[8:8+headerLen], &header); err != nil {
		return errors.New("header is not a JSON object")
	}
	dataLen := uint64(len(content)) - 8 - headerLen
	for key, raw := range header {
		if key == "__metadata__" {
			var metadata map[string]string
			if err := json.Unmarshal(raw, &metadata); err != nil {
				return errors.New("__metadata__ must map strings to strings")
			}
			continue
		}
		var tensor struct {
			Dtype       string   `json:"dtype"`
			Shape       []uint64 `json:"shape"`
			DataOffsets []uint64 `json:"data_offsets"`
		}
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&tensor); err != nil {
			return fmt.Errorf("tensor %q has an invalid description", key)
		}
		size, ok := safetensorsDtypes[tensor.Dtype]
		if !ok {
			return fmt.Errorf("tensor %q has unsupported dtype %q", key, tensor.Dtype)
		}
		if len(tensor.DataOffsets) != 2 || tensor.DataOffsets[0] > tensor.DataOffsets[1] || tensor.DataOffsets[1] > dataLen {
			return fmt.Errorf("tensor %q has invalid data offsets", key)
		}
		elements := uint64(1)
		for _, dim := range tensor.Shape {
			if dim != 0 && elements > (1<<40)/dim {
				return fmt.Errorf("tensor %q shape is too large", key)
			}
			elements *= dim
		}
		if elements*size != tensor.DataOffsets[1]-tensor.DataOffsets[0] {
			return fmt.Errorf("tensor %q size does not match its shape and dtype", key)
		}
	}
	return nil
}
