package service

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const validManifest = `{"name":"Neural","entrypoint":"bot.py","protocol_version":"1.0","runtime":"python-ml-cpu"}`

// zipEntries arma un ZIP con los archivos dados, en orden estable.
func zipEntries(t *testing.T, files [][2]string) []byte {
	t.Helper()
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, file := range files {
		entry, err := writer.Create(file[0])
		require.NoError(t, err)
		_, err = entry.Write([]byte(file[1]))
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	return output.Bytes()
}

// npyBytes arma un .npy v1 con el dtype indicado y datos de relleno.
func npyBytes(descr string, data []byte) []byte {
	header := fmt.Sprintf("{'descr': '%s', 'fortran_order': False, 'shape': (%d,), }", descr, len(data))
	for (10+len(header)+1)%64 != 0 {
		header += " "
	}
	header += "\n"
	out := []byte("\x93NUMPY\x01\x00")
	lenBytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(lenBytes, uint16(len(header)))
	out = append(out, lenBytes...)
	out = append(out, header...)
	return append(out, data...)
}

func npzBytes(t *testing.T, arrays map[string][]byte) string {
	t.Helper()
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for name, content := range arrays {
		entry, err := writer.Create(name)
		require.NoError(t, err)
		_, err = entry.Write(content)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	return output.String()
}

func safetensorsBytes(t *testing.T, header map[string]any, data []byte) string {
	t.Helper()
	raw, err := json.Marshal(header)
	require.NoError(t, err)
	out := make([]byte, 8)
	binary.LittleEndian.PutUint64(out, uint64(len(raw)))
	return string(append(append(out, raw...), data...))
}

func validSafetensors(t *testing.T) string {
	return safetensorsBytes(t, map[string]any{
		"__metadata__": map[string]string{"format": "pt"},
		"w":            map[string]any{"dtype": "F32", "shape": []int{2, 2}, "data_offsets": []int{0, 16}},
	}, make([]byte, 16))
}

func TestReadBotBundle_AcceptsV1AndV2(t *testing.T) {
	r := require.New(t)

	// v1: exactamente agentrix.json + bot.py, sin runtime -> python-stdlib.
	v1, err := readBotBundle(zipEntries(t, [][2]string{
		{"agentrix.json", `{"name":"Classic","entrypoint":"bot.py","protocol_version":"1.0"}`},
		{"bot.py", "print('hi')\n"},
	}))
	r.NoError(err)
	r.Equal(RuntimePythonStdlib, v1.Manifest.Runtime)
	r.Len(v1.Files, 2)

	// v2: módulos extra y los cuatro formatos de modelo.
	files := [][2]string{
		{"agentrix.json", validManifest},
		{"bot.py", "import policy\n"},
		{"policy.py", "WEIGHTS = 'model/weights.npz'\n"},
		{"model/policy.onnx", "\x08\x07onnx-bytes"},
		{"model/weights.safetensors", validSafetensors(t)},
		{"model/weights.npz", npzBytes(t, map[string][]byte{"w.npy": npyBytes("<f4", make([]byte, 8))})},
		{"model/config/normalization.json", `{"mean":[0.0],"std":[1.0]}`},
	}
	v2, err := readBotBundle(zipEntries(t, files))
	r.NoError(err)
	r.Equal(RuntimePythonMLCPU, v2.Manifest.Runtime)
	r.Len(v2.Files, len(files))
	r.Len(v2.Digest, 64)
	r.Len(v2.FileDigests, len(files))

	// El digest no depende del orden de los archivos en el ZIP.
	reversed := make([][2]string, len(files))
	for i, f := range files {
		reversed[len(files)-1-i] = f
	}
	again, err := readBotBundle(zipEntries(t, reversed))
	r.NoError(err)
	r.Equal(v2.Digest, again.Digest)
}

func TestReadBotBundle_RejectsUnsafeOrInvalidContent(t *testing.T) {
	base := [][2]string{{"agentrix.json", validManifest}, {"bot.py", "pass\n"}}
	with := func(extra ...[2]string) [][2]string { return append(append([][2]string{}, base...), extra...) }

	cases := map[string]struct {
		files   [][2]string
		message string
	}{
		"path traversal":             {with([2]string{"../evil.py", "x"}), "unsafe archive path"},
		"absolute path":              {with([2]string{"/etc/passwd", "x"}), "unsafe archive path"},
		"disallowed extension":       {with([2]string{"model/policy.pt", "x"}), "must be .onnx, .safetensors, .npz or .json"},
		"file outside model/":        {with([2]string{"data/weights.npz", "x"}), "model files go under model/"},
		"hidden model segment":       {with([2]string{"model/.secret.json", "{}"}), "invalid path segment"},
		"non-python root file":       {with([2]string{"run.sh", "x"}), "root accepts agentrix.json and .py modules"},
		"invalid json model":         {with([2]string{"model/config.json", "{not json"}), "is not valid JSON"},
		"npz with pickle object":     {with([2]string{"model/w.npz", npzBytes(t, map[string][]byte{"w.npy": npyBytes("|O", nil)})}), "no Python objects"},
		"npz with non-npy entry":     {with([2]string{"model/w.npz", npzBytes(t, map[string][]byte{"payload.pkl": []byte("x")})}), "unexpected entry"},
		"corrupt safetensors header": {with([2]string{"model/w.safetensors", "\xff\xff\xff\xff\xff\xff\xff\xffgarbage"}), "invalid header length"},
		"safetensors size mismatch": {with([2]string{"model/w.safetensors", safetensorsBytes(t, map[string]any{
			"w": map[string]any{"dtype": "F32", "shape": []int{4}, "data_offsets": []int{0, 8}},
		}, make([]byte, 8))}), "size does not match"},
		"unknown runtime":  {[][2]string{{"agentrix.json", `{"name":"x","entrypoint":"bot.py","protocol_version":"1.0","runtime":"gpu"}`}, {"bot.py", "pass\n"}}, "runtime must be"},
		"missing bot.py":   {[][2]string{{"agentrix.json", validManifest}, {"policy.py", "x"}}, "agentrix.json and bot.py are required"},
		"module too large": {with([2]string{"big.py", strings.Repeat("#", maxModuleBytes+1)}), "exceeds"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := readBotBundle(zipEntries(t, tc.files))
			require.ErrorIs(t, err, ErrInvalidBotBundle)
			require.ErrorContains(t, err, tc.message)
		})
	}
}

// ZIP bomb: 30 MiB de un solo byte comprimidos a pocos KB. Se rechaza por
// el tamaño declarado, antes de descomprimirlo.
func TestReadBotBundle_RejectsZipBomb(t *testing.T) {
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, file := range [][2]string{{"agentrix.json", validManifest}, {"bot.py", "pass\n"}} {
		entry, err := writer.Create(file[0])
		require.NoError(t, err)
		_, err = entry.Write([]byte(file[1]))
		require.NoError(t, err)
	}
	entry, err := writer.CreateHeader(&zip.FileHeader{Name: "model/zeros.json", Method: zip.Deflate})
	require.NoError(t, err)
	_, err = entry.Write(bytes.Repeat([]byte(" "), 30<<20))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	require.Less(t, output.Len(), 1<<20, "the bomb itself is small")

	_, err = readBotBundle(output.Bytes())
	require.ErrorIs(t, err, ErrInvalidBotBundle)
	require.ErrorContains(t, err, "exceeds")
}

// Un archivo de modelo muy repetitivo pero dentro de los límites es válido:
// comprimir mucho no lo convierte en una bomba.
func TestReadBotBundle_AcceptsHighlyCompressibleModel(t *testing.T) {
	_, err := readBotBundle(zipEntries(t, [][2]string{
		{"agentrix.json", validManifest}, {"bot.py", "pass\n"},
		{"model/weights.json", "[" + strings.Repeat("0.125,", 700_000) + "0]"},
	}))
	require.NoError(t, err)
}

// Dentro del límite de 50 MB un paquete se rechaza si supera el total.
func TestReadBotBundle_RejectsOversizedModel(t *testing.T) {
	big := strings.Repeat("0", maxModelBytes+1)
	_, err := readBotBundle(zipEntries(t, [][2]string{
		{"agentrix.json", validManifest}, {"bot.py", "pass\n"}, {"model/big.json", big},
	}))
	require.ErrorIs(t, err, ErrInvalidBotBundle)
}
