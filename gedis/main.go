package main

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/muhamadazmy/gedis"
	"github.com/muhamadazmy/gedis/modules/mem"
	"github.com/muhamadazmy/gedis/transport"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func load(mgr gedis.PackageManager, packages string) error {
	pkgs, err := ioutil.ReadDir(packages)
	if err != nil {
		return errors.Wrapf(err, "cannot read packages directory")
	}

	for _, pkg := range pkgs {
		if !pkg.IsDir() {
			continue
		}

		path := filepath.Join(packages, pkg.Name())
		if err := mgr.Add(pkg.Name(), path); err != nil {
			log.Error().Str("name", pkg.Name()).Str("path", path).Err(err).Msg("failed to load package")
		}
	}

	return nil
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{
		TimeFormat: time.RFC3339,
		Out:        os.Stdout,
	})

	var (
		packages string
		address  string
	)

	flag.StringVar(&packages, "packages", "", "path to the packages direcotry")
	flag.StringVar(&address, "address", ":9091", "listen address")

	flag.Parse()

	// we pre load mem.Module, we will probably add more
	// and may be configure modules per package.
	mgr := gedis.NewPackageManager(
		mem.Module, // enable mem module
	)

	if len(packages) != 0 {
		if err := load(mgr, packages); err != nil {
			log.Fatal().Err(err).Msg("failed to load packages")
		}
	}

	// Start the redis transport
	redis := transport.NewRedis(address, mgr)
	log.Info().Msg("listing")
	if err := redis.ListenAndServe(); err != nil {
		log.Fatal().Err(err).Msg("failed to start redis transport")
	}

}
