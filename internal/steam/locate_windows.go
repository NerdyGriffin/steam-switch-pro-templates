//go:build windows

package steam

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

// Locate finds the Steam install on Windows.
//
// Lookup order:
//  1. HKCU\Software\Valve\Steam\SteamPath (per-user, set by the Steam client)
//  2. HKLM\Software\Wow6432Node\Valve\Steam\InstallPath (system-wide, set by installer)
//  3. HKLM\Software\Valve\Steam\InstallPath (64-bit fallback)
func Locate() (*Install, error) {
	candidates := []func() (string, error){
		readHKCUSteamPath,
		readHKLMInstallPath,
		readHKLM64InstallPath,
	}
	for _, fn := range candidates {
		path, err := fn()
		if err != nil || path == "" {
			continue
		}
		path = filepath.Clean(path)
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return &Install{Root: path}, nil
		}
	}
	return nil, ErrSteamNotFound
}

func readHKCUSteamPath() (string, error) {
	return readRegString(registry.CURRENT_USER, `Software\Valve\Steam`, "SteamPath")
}

func readHKLMInstallPath() (string, error) {
	return readRegString(registry.LOCAL_MACHINE, `SOFTWARE\Wow6432Node\Valve\Steam`, "InstallPath")
}

func readHKLM64InstallPath() (string, error) {
	return readRegString(registry.LOCAL_MACHINE, `SOFTWARE\Valve\Steam`, "InstallPath")
}

func readRegString(root registry.Key, path, name string) (string, error) {
	k, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", path, err)
	}
	defer k.Close()
	val, _, err := k.GetStringValue(name)
	if err != nil {
		return "", fmt.Errorf("read %s\\%s: %w", path, name, err)
	}
	return val, nil
}
