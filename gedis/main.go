package main

import (
	"fmt"

	"github.com/muhamadazmy/gedis"
	lua "github.com/yuin/gopher-lua"
)

func main() {
	pkg, err := gedis.NewPackage("../examples/pkg")
	if err != nil {
		panic(err)
	}

	// pool := gedis.NewPool(10)
	// fmt.Println(pool)
	state, err := pkg.Get()
	if err != nil {
		panic(err)
	}

	// fmt.Println("After loading Top:", state.GetTop())
	fn := state.GetGlobal("add")
	//fmt.Println(prt.Type())
	state.Push(fn)
	state.Push(lua.LNumber(10))
	state.Push(lua.LNumber(20))
	state.Call(2, lua.MultRet)
	fmt.Println(state.ToString(-1))
	state.Pop(1)
	state.Close()

	// fmt.Println(pool)
	// pool.Close()
	// fmt.Println(pool)

	// // lua.LTFunction
	// err := state.DoString(`print("hello", "world")`)
	// if err != nil {
	// 	panic(err)
	// }
}
