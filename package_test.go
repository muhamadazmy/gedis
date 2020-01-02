package gedis

import (
	"fmt"
	"github.com/stretchr/testify/require"
	lua "github.com/yuin/gopher-lua"
	"testing"
)

func TestValue(t *testing.T) {
	require := require.New(t)
	pool := NewPool(1)
	defer pool.Close()
	l, err := pool.Get()
	require.NoError(err)
	defer l.Close()

	cases := []struct {
		value interface{}
		typ   lua.LValueType
	}{
		{10, lua.LTNumber},
		{uint8(200), lua.LTNumber},
		{"hello world", lua.LTString},
		{true, lua.LTBool},
		{false, lua.LTBool},
		{nil, lua.LTNil},
		{[]int{1, 2, 3}, lua.LTTable},
		{struct{ V int }{1}, lua.LTTable},
	}

	for _, c := range cases {
		t.Run(fmt.Sprint(c.value), func(t *testing.T) {
			v := value(l, c.value)
			require.NotNil(v)
			require.Equal(c.typ, v.Type())
		})
	}
}

func TestTableValue(t *testing.T) {
	require := require.New(t)
	pool := NewPool(1)
	defer pool.Close()
	l, err := pool.Get()
	require.NoError(err)
	defer l.Close()

	s := struct {
		Name      string
		Age       float64
		Recursive struct {
			Sub string
		}
		Values  []float64
		Map     map[string]string
		ignored bool
	}{
		Name:   "test",
		Age:    100,
		Values: []float64{1, 2, 3},
		Map: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	s.Recursive.Sub = "sub value"

	v := value(l, s)
	require.NotNil(v)
	require.Equal(lua.LTTable, v.Type())

	l.SetGlobal("data", v)

	code := `
assert(data.name == "test", "invalid name")
assert(data.age == 100, "invalid age")
assert(data.map.key1 == "value1", "invalid value key1 in map")
assert(data.recursive.sub == "sub value", "invalid value in sub struct")
assert(data.values[1] == 1, "invalid array")
`
	require.NoError(l.DoString(code))
}
