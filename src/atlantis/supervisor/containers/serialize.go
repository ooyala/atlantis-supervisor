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

package containers

import (
	"encoding/json"
	"log"
	"os"
	"path"
)

func save() {
	saveObject(ContainersFile, containers)
	saveObject(PortsFile, ports)
}

// Use json to save an object to a file
func saveObject(file string, object interface{}) {
	fo, err := os.Create(path.Join(SaveDir, file))
	if err != nil {
		log.Printf("Could not save %s: %s", file, err)
		// hope everything works out.
		// TODO[jigish] email error
		return
	}
	defer fo.Close()
	e := json.NewEncoder(fo)
	if err := e.Encode(object); err != nil {
		log.Println("ERROR: cannot save object " + file + ": " + err.Error())
	}
}

// Use json to retrieve an object from a file
func retrieveObject(file string, object interface{}) bool {
	fi, err := os.Open(path.Join(SaveDir, file))
	if err != nil {
		log.Printf("Could not retrieve %s: %s", file, err)
		return false
	}
	d := json.NewDecoder(fi)
	if err := d.Decode(object); err != nil {
		log.Println("ERROR: could not retrieve object " + file + ": " + err.Error())
	}
	log.Printf("Retrieved %s: %#v", file, object)
	if object == nil {
		log.Println("Object retrieved was nil.")
		return false
	}
	return true
}
