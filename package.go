package gedis

import (
	"bufio"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	lua "github.com/yuin/gopher-lua"
	"github.com/yuin/gopher-lua/parse"
)

// Package represents a gedis package
type Package struct {
	pool *StatePool
}

// CompileLua reads the passed lua file from disk and compiles it.
func compileLua(filePath string) (*lua.FunctionProto, error) {
	file, err := os.Open(filePath)
	defer file.Close()
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(file)
	chunk, err := parse.Parse(reader, filePath)
	if err != nil {
		return nil, err
	}
	proto, err := lua.Compile(chunk, filePath)
	if err != nil {
		return nil, err
	}
	return proto, nil
}

// NewPackage create a new package
func NewPackage(p string) (*Package, error) {
	files, err := ioutil.ReadDir(p)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list package directory")
	}

	var protos []*lua.FunctionProto
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		path := filepath.Join(p, file.Name())
		proto, err := compileLua(path)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compile file: %s", path)
		}
		protos = append(protos, proto)
	}

	pool := NewPool(50, PoolOptions{
		Open: func() (*lua.LState, error) {
			state := lua.NewState()
			for _, proto := range protos {
				fn := state.NewFunctionFromProto(proto)
				state.Push(fn)
				if err := state.PCall(0, lua.MultRet, nil); err != nil {
					return nil, errors.Wrap(err, "failed to preload package files")
				}
			}

			return state, nil
		},
	})

	pkg := &Package{
		pool: pool,
	}

	return pkg, nil
}

// Get a Lua State object
func (p *Package) Get() (*LState, error) {
	return p.pool.Get()
}

// Call a package method
func (p *Package) Call(fn string, args ...interface{}) error {
	L, err := p.pool.Get()
	if err != nil {
		return err
	}
	defer L.Close()

	// TODO: forward call to fn
	// 1- Build the proper L Values
	return nil
}
