package model

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// Estructura en disco de un paquete de bot v2 (ADR-0014):
//
//	submissions/<agente>/v<n>/bundle/bot.py   <- code_path
//	submissions/<agente>/v<n>/bundle/...      <- resto del paquete
//	submissions/<agente>/v<n>/bundle-manifest.json
const (
	BotBundleDirName      = "bundle"
	BotBundleManifestName = "bundle-manifest.json"
)

// BotBundleDir devuelve la carpeta del paquete v2 que contiene codePath, o
// ok=false si codePath es un bot v1 de un solo archivo.
func BotBundleDir(codePath string) (dir string, ok bool) {
	dir = filepath.Dir(codePath)
	if filepath.Base(dir) != BotBundleDirName {
		return "", false
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dir), BotBundleManifestName)); err != nil {
		return "", false
	}
	return dir, true
}

// BundleListingDigest es el digest de un paquete: SHA-256 del listado
// ordenado "ruta\x00sha256\n" de todos sus archivos. No depende del orden
// de los archivos en el ZIP ni del sistema de archivos.
func BundleListingDigest(fileDigests map[string]string) string {
	paths := make([]string, 0, len(fileDigests))
	for name := range fileDigests {
		paths = append(paths, name)
	}
	sort.Strings(paths)
	listing := sha256.New()
	for _, name := range paths {
		fmt.Fprintf(listing, "%s\x00%s\n", name, fileDigests[name])
	}
	return hex.EncodeToString(listing.Sum(nil))
}

// BotArtifactDigest identifica el código que ejecuta un slot: el digest
// del paquete completo, recalculado desde el disco, para un bot v2, o el
// SHA-256 del archivo para un bot v1.
func BotArtifactDigest(codePath string) (string, error) {
	dir, ok := BotBundleDir(codePath)
	if !ok {
		return ComputeFileSHA256(codePath)
	}
	digests := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		digest, err := ComputeFileSHA256(path)
		if err != nil {
			return err
		}
		digests[filepath.ToSlash(rel)] = digest
		return nil
	})
	if err != nil {
		return "", err
	}
	return BundleListingDigest(digests), nil
}
