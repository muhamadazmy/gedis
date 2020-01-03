package gedis

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"unicode"

	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
	"github.com/yuin/gluamapper"
	lua "github.com/yuin/gopher-lua"
	"github.com/yuin/gopher-lua/parse"
)

var (
	defaultMapOption = gluamapper.Option{}
)

// Results are call results
type Results []interface{}

// Module definition
type Module func(*lua.LState) (string, lua.LGFunction)

// Package interface
type Package interface {
	Call(fn string, args ...interface{}) (Results, error)
	Close() error
}

// luaPackage represents a gedis package
type luaPackage struct {
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
func NewPackage(p string, modules ...Module) (Package, error) {
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

			// preload modules
			for _, module := range modules {
				state.PreloadModule(module(state))
			}

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

	pkg := &luaPackage{
		pool: pool,
	}

	return pkg, nil
}

func (p *luaPackage) Close() error {
	return p.pool.Close()
}

// Call a package method
func (p *luaPackage) Call(fn string, args ...interface{}) (Results, error) {
	L, err := p.pool.Get()
	if err != nil {
		return nil, err
	}
	defer L.Close()

	fnValue := L.GetGlobal(fn)

	if fnValue.Type() == lua.LTNil {
		return nil, fmt.Errorf("unknown function")
	}
	L.Push(fnValue)
	for _, arg := range args {
		L.Push(value(L, arg))
	}

	err = L.PCall(len(args), lua.MultRet, nil)
	if err != nil {
		return nil, err
	}
	results := make([]lua.LValue, L.GetTop())

	top := L.GetTop()
	for i := -top; i < 0; i++ {
		results[len(results)+i] = L.Get(i)
	}
	L.Pop(L.GetTop())

	var returns []interface{}
	for _, result := range results {

		returns = append(returns,
			gluamapper.ToGoValue(result, defaultMapOption),
		)
	}

	return returns, nil
}

func fromNumber(in interface{}) lua.LValue {
	switch v := in.(type) {
	case int:
		return lua.LNumber(v)
	case int8:
		return lua.LNumber(v)
	case int16:
		return lua.LNumber(v)
	case int32:
		return lua.LNumber(v)
	case int64:
		return lua.LNumber(v)
	case uint:
		return lua.LNumber(v)
	case uint8:
		return lua.LNumber(v)
	case uint16:
		return lua.LNumber(v)
	case uint32:
		return lua.LNumber(v)
	case uint64:
		return lua.LNumber(v)
	case float32:
		return lua.LNumber(v)
	case float64:
		return lua.LNumber(v)
	default:
		return nil
	}

}

func fromData(l *LState, in interface{}) lua.LValue {
	t := l.NewTable()
	//t.RawGetH()
	v := reflect.ValueOf(in)
	switch v.Kind() {
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			t.Append(value(l, v.Index(i).Interface()))
		}
	case reflect.Map:
		r := v.MapRange()
		for r.Next() {
			t.RawSet(value(l, r.Key().Interface()), value(l, r.Value().Interface()))
		}
	case reflect.Struct:
		typ := v.Type()
		for i := 0; i < typ.NumField(); i++ {
			name := typ.Field(i).Name
			var f rune
			// TODO find a better way to get
			// the first rune of a string
			// NOTE: we do this not with 'index' because unicode rune
			// might span more than one byte. hence iteration is the
			// only way to read a full rune
			for _, c := range name {
				f = c
				break
			}

			if unicode.IsLower(f) {
				continue
			}
			t.RawSetString(strcase.ToSnake(name), value(l, v.Field(i).Interface()))
		}
	//TODO: case pointers: we probably never have this use case.
	default:
		panic(fmt.Sprintf("invald kind '%s'", v.Kind()))
	}

	return t
}

//value return a lua value from Go value.
//TODO: cover entire range of builtin values, plus tables, arrays and structures
func value(l *LState, in interface{}) lua.LValue {
	num := fromNumber(in)
	if num != nil {
		return num
	}

	switch v := in.(type) {
	case nil:
		return lua.LNil
	case bool:
		return lua.LBool(v)
	case string:
		return lua.LString(v)
	default:
		return fromData(l, in)
	}
}
