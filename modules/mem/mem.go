package mem

import (
	lua "github.com/yuin/gopher-lua"
	"sync"
)

var (
	cache = map[string]lua.LValue{}
	m     sync.RWMutex
)

// Module is module entry point
func Module(L *lua.LState) (string, lua.LGFunction) {
	return "mem", loader
}

// loader for mem module
func loader(L *lua.LState) int {
	// register functions to the table
	mod := L.SetFuncs(L.NewTable(), exports)

	// returns the module
	L.Push(mod)
	return 1
}

var exports = map[string]lua.LGFunction{
	"set": set,
	"get": get,
}

func set(L *lua.LState) int {
	key := L.CheckString(1)
	value := L.CheckAny(2)

	m.Lock()
	defer m.Unlock()

	cache[key] = value
	return 0
}

func get(L *lua.LState) int {
	key := L.CheckString(1)

	m.RLock()
	defer m.RUnlock()

	value, ok := cache[key]
	if !ok {
		L.Push(lua.LNil)
	} else {
		L.Push(value)
	}

	return 1
}
