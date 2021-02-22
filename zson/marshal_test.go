package zson_test

/* NOT YET

func trim(s string) string {
	return strings.TrimSpace(s) + "\n"
}

type Thing interface {
	Color() string
}

type Plant struct {
	MyColor string
}

func (p *Plant) Color() string { return p.MyColor }

type Animal struct {
	MyColor string
}

func (a *Animal) Color() string { return a.MyColor }

func TestInterfaceMarshal(t *testing.T) {
	rose := Thing(&Plant{"red"})
	expectedRose := `{MyColor:"red"} (=Plant)`
	flamingo := Thing(&Animal{"pink"})
	expectedFlamingo := `{MyColor:"pink"} (=Animal)`

	m := zson.NewMarshaler()
	m.Decorate(zson.StyleSimple)

	zsonRose, err := m.Marshal(rose)
	require.NoError(t, err)
	assert.Equal(t, trim(expectedRose), trim(zsonRose))

	zsonFlamingo, err := m.Marshal(flamingo)
	require.NoError(t, err)
	assert.Equal(t, trim(expectedFlamingo), trim(zsonFlamingo))

	u := zson.NewUnmarshaler()
	u.Bind(Plant{}, Animal{})
	var thing Thing
	require.NoError(t, err)

	err = u.Unmarshal(zsonRose, &thing)
	require.NoError(t, err)
	assert.Equal(t, "red", thing.Color())

	err = u.Unmarshal(zsonFlamingo, &thing)
	require.NoError(t, err)
	assert.Equal(t, "pink", thing.Color())
}

type Roll bool

func TestMarshal(t *testing.T) {
	z, err := zson.Marshal("hello, world")
	require.NoError(t, err)
	assert.Equal(t, `"hello, world"`, z)

	aIn := []int8{1, 2, 3}
	z, err = zson.Marshal(aIn)
	require.NoError(t, err)
	assert.Equal(t, `[1 (int8),2 (int8),3 (int8)] (=0)`, z)

	var v interface{}
	err = zson.Unmarshal(z, &v)
	require.NoError(t, err)
	aOut, ok := v.([]int8)
	assert.Equal(t, ok, true)
	assert.Equal(t, aIn, aOut)

	m := zson.NewMarshaler()
	m.Decorate(zson.StyleSimple)
	z, err = m.Marshal(Roll(true))
	require.NoError(t, err)
	assert.Equal(t, `true (=Roll)`, z)
}

*/
