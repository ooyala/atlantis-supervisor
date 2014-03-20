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
	"encoding/json"
	"os"
	"path"
)

var SaveDir string

func Init(saveDir string) error {
	SaveDir = saveDir
	err := os.MkdirAll(SaveDir, 0755)
	if err != nil {
		return err
	}
	return nil
}

type SaveDefinition struct {
	File   string
	Object interface{}
}

func SaveAll(defs ...SaveDefinition) {
	for _, def := range defs {
		SaveObject(def.File, def.Object)
	}
}

// Use json to save an object to a file
func SaveObject(file string, object interface{}) error {
	fo, err := os.Create(path.Join(SaveDir, file))
	if err != nil {
		return err
	}
	defer fo.Close()
	e := json.NewEncoder(fo)
	if err := e.Encode(object); err != nil {
		return err
	}
	return nil
}

// Use json to retrieve an object from a file
func RetrieveObject(file string, object interface{}) error {
	fi, err := os.Open(path.Join(SaveDir, file))
	if err != nil {
		return err
	}
	d := json.NewDecoder(fi)
	if err := d.Decode(object); err != nil {
		return err
	}
	return nil
}
