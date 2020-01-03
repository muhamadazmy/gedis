# Gedis
A proof of concept application server that can serve lua applications. It uses [gopher-lua](https://github.com/yuin/gopher-lua)

# What is done
- [x] Load lua code
- [x] Proxy calls from Go to lua
- [x] Build Go packages/functions that is callable from Lua packages
- [x] Build a RESP (redis) transport so remote clients can call lua functions
- [x] Basic control resp command to list, load, and unload lua packages
- [ ] Client sets `content-type` current only `JSON`
- [ ] Test support for native lua libraries
- [ ] Authentication/Authorization
- [ ] Package manifest to configure
  - [ ] Authentication/Authorization per package (action?)
  - [ ] Go packages to be available from Lua package

## Resp Transport
While the code base supports replacing and or add support to multiple transport layers, currently only Resp transport is implemented. It works as following

- Resp expose built in functions, and packages functions
- All built it functions are prefixed with `.`, and they are case insensitive
- Currently the build in functions are
  - `.ping` response is always `PONG`
  - `.package.list` list names of loaded packages
  - `.package.add <name> <path>` adds lua package add path `path` with name `name`
  - `.package.remove <name>` remove packages `name`
  - `.content-type.get` gets current content type (currently only return `JSON`)
  - `.content-type.set` sets content type for package method calls inputs and outputs (current only return error `not implemented`)
- Package functions calls
  - Package functions calls has to be in format `<package>.<function> args...`
  - arguments must be valid value for selected `content-type` (currently json)
  - Note: the following examples are done with `redis-cli -p 9091`
  - for example a call to `add` function of the `calc` package must be done as `calc.add 1 2` notice that inputs are not quoted because that's how a valid json number is. The return value is always a valid json value. You need to unmarshal the returned data to get a valid type.
  - for a bit of a more complex example, the `db` package
    - `db.set` and `db.get` always require a string key
    - `db.set '"key"' '"string value"` sets value to string value
    - `db.set '"key"' 123` sets value to numeric value
    - `db.get '"key"'` returns the last set value to `key`
    - Note here a valid json string `key` is `"key"`

# Running the examples
In one terminal do
```bash
make run
```
In another terminal do
```bash
redis-cli -p 9091
```

Then in the redis-cli shell, do:
```bash
> .ping
PONG
```

List loaded packages
```bash
> .package.list
1) calc
2) db
```

Make a call
```bash
> calc.add 1 2
"3"
> db.set '"name"' '"test"'
OK
> db.set '"age"' 37
OK
> db.get '"name"'
"\"test\""
> db.get '"age"'
"37
```

The escape sequence is added by `redis-cli` for display purpose, with a redis client, you will receive a valid json string that you can unmarshal normally
