package gedis

import (
	"fmt"
	"github.com/pkg/errors"
	"sync"
)

// PackageManager interface
type PackageManager interface {
	// Call a package function with given arguments
	Call(pkg, fn string, args ...interface{}) (Results, error)
	// Add loads and register a new package with given name and code directory
	Add(name, path string) error
	// Remove a package from manager, the package won't be callable afterwards
	Remove(name string) error
	// List available package names
	List() []string
}

// luaPackageManager manage package loading/unloading
type luaPackageManager struct {
	packages map[string]Package
	modules  []Module
	m        sync.RWMutex
}

// NewPackageManager creates a new package manager
func NewPackageManager(modules ...Module) PackageManager {
	return &luaPackageManager{
		packages: make(map[string]Package),
		modules:  modules,
	}
}

func (m *luaPackageManager) Call(pkg, fn string, args ...interface{}) (Results, error) {
	m.m.RLock()
	defer m.m.RUnlock()

	p, ok := m.packages[pkg]
	if !ok {
		return nil, fmt.Errorf("unknown package '%s'", pkg)
	}

	return p.Call(fn, args...)
}

func (m *luaPackageManager) Add(name, path string) error {
	m.m.Lock()
	defer m.m.Unlock()

	if _, ok := m.packages[name]; ok {
		return fmt.Errorf("package with name '%s' already exists", name)
	}

	pkg, err := NewPackage(path, m.modules...)
	if err != nil {
		return errors.Wrapf(err, "failed to load package: %s (%s)", name, path)
	}

	m.packages[name] = pkg
	return nil
}

func (m *luaPackageManager) Remove(name string) error {
	m.m.Lock()
	defer m.m.Unlock()

	pkg, ok := m.packages[name]
	if !ok {
		return nil
	}

	if err := pkg.Close(); err != nil {
		return err
	}

	delete(m.packages, name)
	return nil
}

func (m *luaPackageManager) List() []string {
	m.m.RLock()
	defer m.m.RUnlock()

	var pkgs []string
	for k := range m.packages {
		pkgs = append(pkgs, k)
	}

	return pkgs
}
