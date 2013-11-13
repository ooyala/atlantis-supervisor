package containers

import (
	"launchpad.net/gocheck"
	"os"
	"testing"
)

func TestSerialize(t *testing.T) { gocheck.TestingT(t) }

type SerializeSuite struct{}

var _ = gocheck.Suite(&SerializeSuite{})

type TestSerializeStruct struct {
	Int    int
	Bool   bool
	String string
	Slice  []string
	Map    map[string]string
}

func (s *SerializeSuite) TestSerialize(c *gocheck.C) {
	SaveDir = "save_test"
	os.RemoveAll(SaveDir)
	c.Assert(os.MkdirAll(SaveDir, 0755), gocheck.IsNil)
	// test save/retrieve slice
	savedSlice := []uint16{5, 4, 3, 2, 1}
	var retrievedSlice []uint16
	saveObject("slice", savedSlice)
	c.Assert(retrieveObject("slice", &retrievedSlice), gocheck.Equals, true)
	c.Assert(retrievedSlice, gocheck.DeepEquals, savedSlice)
	// test save/retrieve map
	savedMap := map[string]*TestSerializeStruct{}
	var retrievedMap map[string]*TestSerializeStruct
	savedMap["one"] = &TestSerializeStruct{1, true, "one", []string{"one", "alsoOne"}, map[string]string{
		"one":     "yes",
		"alsoOne": "alsoYes",
	}}
	savedMap["two"] = &TestSerializeStruct{2, false, "two", nil, nil}
	saveObject("map", savedMap)
	c.Assert(retrieveObject("map", &retrievedMap), gocheck.Equals, true)
	c.Assert(retrievedMap, gocheck.DeepEquals, savedMap)
	os.RemoveAll(SaveDir)
}
