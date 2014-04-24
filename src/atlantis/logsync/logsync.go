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

package logsync

import (
	"crypto/md5"
	"errors"
	"fmt"
	"github.com/crowdmob/goamz/aws"
	"github.com/crowdmob/goamz/s3"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type SyncT struct {
	Dir     string
	Suffix  string
	Bucket  *s3.Bucket
	Prefix  string
	Threads uint
	Auth    aws.Auth
	Dry     bool
	NoAws   bool
	Debug   bool
}

type empty struct{}

type transfer struct {
	Src  string
	Dest string
}

func (s *SyncT) validateDest() error {
	testPath := fmt.Sprintf("%s/sync_test", s.Prefix)
	err := s.Bucket.Put(testPath, []byte("test"), "text", s3.Private, s3.Options{})
	if err != nil {
		return err
	}
	err = s.Bucket.Del(testPath)
	return err
}

func (s *SyncT) validateSrc() error {
	_, err := os.Stat(s.Dir)
	return err
}

func (s *SyncT) validate() error {
	if s.Threads == 0 {
		return errors.New("Number of threads specified must be non-zero")
	}

	err := s.validateSrc()
	if err != nil {
		return err
	}

	if !s.NoAws {
		err = s.validateDest()
		if err != nil {
			return err
		}
	}

	return nil
}

func relativePath(path string, logPath string) string {
	if path == "." {
		return strings.TrimLeft(logPath, "/")
	} else {
		return strings.TrimLeft(strings.TrimPrefix(logPath, path), "/")
	}
}

func (s *SyncT) loadSrc() map[string]string {
	logs := map[string]string{}
	filepath.Walk(s.Dir, func(logPath string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(info.Name(), s.Suffix) {
			path := relativePath(s.Dir, logPath)

			buf, err := ioutil.ReadFile(logPath)
			if err != nil {
				// Something went wrong reading a log, so log and exit
				log.Fatal(err)
			}

			md5Hash := md5.New()
			md5Hash.Write(buf)
			md5sum := fmt.Sprintf("%x", md5Hash.Sum(nil))
			logs[path] = md5sum
			if s.Debug {
				log.Printf("Loading src: %s md5: %s\n", logPath, md5sum)
			}
		}
		return nil
	})
	return logs
}

func (s *SyncT) loadDest() (map[string]string, error) {
	logs := map[string]string{}
	data, err := s.Bucket.List(s.Prefix, "", "", 0)
	if err != nil {
		return nil, err
	}
	if data.IsTruncated == true {
		msg := "Results from S3 truncated and I don't yet know how to download next set of results, so we will exit to avoid invalidating results."
		return nil, errors.New(msg)
	}
	for i := range data.Contents {
		md5sum := strings.Trim(data.Contents[i].ETag, "\"")
		path := relativePath(s.Prefix, data.Contents[i].Key)
		logs[path] = md5sum
		if s.Debug {
			log.Printf("Loading dest: %s md5: %s\n", data.Contents[i].Key, md5sum)
		}
	}
	return logs, nil
}

func putLog(t transfer, bucket *s3.Bucket, dry bool) {
	data, err := ioutil.ReadFile(t.Src)
	if err != nil {
		// Error reading log
		log.Printf("Error reading source file %s:\n", t.Src)
		log.Fatal(err)
	}

	contType := "binary/octet-stream"
	perm := s3.ACL("private")

	if dry {
		log.Printf("Starting sync of %s to bucket path %s...\n", t.Src, t.Dest)
	} else {
		log.Printf("Starting sync of %s to s3://%s/%s...\n", t.Src, bucket.Name, t.Dest)
		err = bucket.Put(t.Dest, data, contType, perm, s3.Options{})
		if err != nil {
			// Error uploading log to s3
			log.Printf("Sync of %s to s3://%s/%s failed:\n", t.Src, bucket.Name, t.Dest)
			log.Fatal(err)
		}
	}
}

func syncFile(t transfer, bucket *s3.Bucket, workerChan chan empty, wg *sync.WaitGroup, dry bool) {
	defer wg.Done()
	putLog(t, bucket, dry)
	<-workerChan
}

func workerSpawner(bucket *s3.Bucket, fileChan chan transfer, workerChan chan empty, dieChan chan empty, wg *sync.WaitGroup, dry bool) {
	for {
		select {
		case file := <-fileChan:
			wg.Add(1)
			go syncFile(file, bucket, workerChan, wg, dry)
		case <-dieChan:
			return
		}
	}
}

func (s *SyncT) syncLogs(src, dest map[string]string) error {
	fileChan := make(chan transfer)
	workerChan := make(chan empty, s.Threads)
	dieChan := make(chan empty)
	var wg sync.WaitGroup
	go workerSpawner(s.Bucket, fileChan, workerChan, dieChan, &wg, s.Dry)
	for f, _ := range src {
		if dest[f] != src[f] {
			srcPath := strings.Join([]string{s.Dir, f}, "/")
			destPath := strings.Join([]string{s.Prefix, f}, "/")
			workerChan <- empty{}
			fileChan <- transfer{srcPath, destPath}
		}
	}
	wg.Wait()
	dieChan <- empty{}
	return nil
}

func (s *SyncT) Sync() error {
	err := s.validate()
	if err != nil {
		return err
	}

	var destLogs map[string]string
	if !s.NoAws {
		destLogs, err = s.loadDest()
		if err != nil {
			return err
		}
	}
	srcLogs := s.loadSrc()

	return s.syncLogs(srcLogs, destLogs)
}
