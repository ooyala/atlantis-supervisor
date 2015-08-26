/* Copyright 2014 Ooyala, Inc. All rights reserved.
 *
 * This file is licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
 * except in compliance with the License. You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License is
 * distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and limitations under the License.
 */

package serialize

import (
	"github.com/adjust/gocheck"
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
	SaveObject("slice", savedSlice)
	c.Assert(RetrieveObject("slice", &retrievedSlice), gocheck.Equals, nil)
	c.Assert(retrievedSlice, gocheck.DeepEquals, savedSlice)
	// test save/retrieve map
	savedMap := map[string]*TestSerializeStruct{}
	var retrievedMap map[string]*TestSerializeStruct
	savedMap["one"] = &TestSerializeStruct{1, true, "one", []string{"one", "alsoOne"}, map[string]string{
		"one":     "yes",
		"alsoOne": "alsoYes",
	}}
	savedMap["two"] = &TestSerializeStruct{2, false, "two", nil, nil}
	SaveObject("map", savedMap)
	c.Assert(RetrieveObject("map", &retrievedMap), gocheck.Equals, nil)
	c.Assert(retrievedMap, gocheck.DeepEquals, savedMap)
	os.RemoveAll(SaveDir)
}
