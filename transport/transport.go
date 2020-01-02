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
		fallthrough
	case ".package.remove":
		fallthrough
	case ".set.content-type":
		conn.WriteString("OK")
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
