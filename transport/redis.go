package transport

import (
	"encoding/json"
	"fmt"
	"github.com/muhamadazmy/gedis"
	"github.com/pkg/errors"
	"github.com/tidwall/redcon"
	"strings"
)

// Redis transport
type Redis struct {
	mgr  gedis.PackageManager
	addr string
}

// NewRedis creates a new redis stransport
func NewRedis(addr string, mgr gedis.PackageManager) *Redis {
	return &Redis{addr: addr, mgr: mgr}
}

func (r *Redis) packageRemove(cmd redcon.Command) error {
	args := cmd.Args[1:]
	if len(args) != 1 {
		return fmt.Errorf("invalid number of arguments expecting '<name>'")
	}

	name := string(args[0])
	return r.mgr.Remove(name)
}

func (r *Redis) packageAdd(cmd redcon.Command) error {
	args := cmd.Args[1:]
	if len(args) != 2 {
		return fmt.Errorf("invalid number of arguments expecting '<name> <path>'")
	}

	name := string(args[0])
	path := string(args[1])
	return r.mgr.Add(name, path)
}

func (r *Redis) simple(conn redcon.Conn, cmd redcon.Command, f func(cmd redcon.Command) error) {
	err := f(cmd)
	if err != nil {
		conn.WriteError(err.Error())
		return
	}

	conn.WriteString("OK")
}

func (r *Redis) handleInternal(conn redcon.Conn, cmd redcon.Command) {
	// we allow internal commands to be case insensitive
	c := strings.ToLower(string(cmd.Args[0]))
	switch c {
	case ".ping":
		conn.WriteString("PONG")
	case ".package.list":
		pkgs := r.mgr.List()
		conn.WriteArray(len(pkgs))
		for _, pkg := range pkgs {
			conn.WriteString(pkg)
		}
	case ".package.add":
		r.simple(conn, cmd, r.packageAdd)
	case ".package.remove":
		r.simple(conn, cmd, r.packageRemove)
	case ".content-type.set":
		conn.WriteError("not implemented")
	case ".content-type.get":
		conn.WriteString("JSON")
	default:
		conn.WriteError(fmt.Sprintf("unknown command '%s'", c))
	}
}

func (r *Redis) handle(conn redcon.Conn, cmd redcon.Command) {
	c := string(cmd.Args[0])
	if strings.HasPrefix(c, ".") {
		// internal command
		r.handleInternal(conn, cmd)
		return
	}
	// call package.
	parts := strings.SplitN(c, ".", 2)
	if len(parts) != 2 {
		conn.WriteError(fmt.Sprintf("invalid command name format expecting <package>.<action> got '%s'", c))
		return
	}

	var args []interface{}
	for i, input := range cmd.Args[1:] {
		var arg interface{}
		// TODO: respect content-type associated with the connection
		if err := json.Unmarshal(input, &arg); err != nil {
			conn.WriteError(errors.Wrapf(err, "failed to load argument '%d'", i).Error())
			return
		}
		args = append(args, arg)
	}

	// make a call
	results, err := r.mgr.Call(parts[0], parts[1], args...)
	if err != nil {
		conn.WriteError(err.Error())
		return
	}

	// if the action return just one result, we need to
	// return this as a single value (not list)
	if len(results) == 0 {
		conn.WriteString("OK")
		return
	} else if len(results) == 1 {
		r.writeResult(conn, results[0])
		return
	}

	// if the call returns many results (tuple)
	conn.WriteArray(len(results))
	for _, result := range results {
		r.writeResult(conn, result)
	}
}

func (r *Redis) writeResult(conn redcon.Conn, o interface{}) {
	data, err := json.Marshal(o)
	if err != nil {
		conn.WriteError(err.Error())
	} else {
		conn.WriteBulk(data)
	}
}

func (r *Redis) accept(conn redcon.Conn) bool {
	return true
}

func (r *Redis) closed(conn redcon.Conn, err error) {

}

// ListenAndServe start redis transport
func (r *Redis) ListenAndServe() error {
	return redcon.ListenAndServe(r.addr, r.handle, r.accept, r.closed)
}
