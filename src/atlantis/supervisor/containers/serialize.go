package containers

import (
	"bufio"
	"encoding/gob"
	"log"
	"os"
	"path"
)

func save() {
	saveObject(ContainersFile, containers)
	saveObject(PortsFile, ports)
	if Proxy != nil {
		saveObject(ProxyFile, Proxy)
	}
}

// Use gob to save an object to a file
func saveObject(file string, object interface{}) {
	gob.Register(object)
	fo, err := os.Create(path.Join(SaveDir, file))
	if err != nil {
		log.Printf("Could not save %s: %s", file, err)
		// hope everything works out.
		// TODO[jigish] email error
		return
	}
	defer fo.Close()
	w := bufio.NewWriter(fo)
	e := gob.NewEncoder(w)
	e.Encode(object)
	w.Flush()
}

// Use gob to retrieve an object from a file
func retrieveObject(file string, object interface{}) bool {
	fi, err := os.Open(path.Join(SaveDir, file))
	if err != nil {
		log.Printf("Could not retrieve %s: %s", file, err)
		return false
	}
	r := bufio.NewReader(fi)
	d := gob.NewDecoder(r)
	d.Decode(object)
	log.Printf("Retrieved %s: %#v", file, object)
	return true
}
