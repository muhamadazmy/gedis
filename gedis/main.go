package main

import (
	"fmt"
	"github.com/muhamadazmy/gedis"
	"github.com/muhamadazmy/gedis/modules/mem"
	"github.com/muhamadazmy/gedis/transport"
)

func main() {
	// make mem module available for packages.

	mgr := gedis.NewPackageManager(mem.Module)
	if err := mgr.Add("calc", "../examples/pkg"); err != nil {
		panic(err)
	}

	if err := mgr.Add("db", "../examples/db"); err != nil {
		panic(err)
	}

	redis := transport.NewRedis(":9090", mgr)
	fmt.Println("listening")
	if err := redis.ListenAndServe(); err != nil {
		panic(err)
	}

}
