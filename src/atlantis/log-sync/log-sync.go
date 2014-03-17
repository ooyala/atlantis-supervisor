package logsync

import (
	"crypto/md5"
	"errors"
	"fmt"
	"github.com/crowdmob/goamz/aws"
	"github.com/crowdmob/goamz/s3"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
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
				// Something went wrong reading a log, so print and panic
				return err
			}

			md5Hash := md5.New()
			md5Hash.Write(buf)
			md5sum := fmt.Sprintf("%x", md5Hash.Sum(nil))
			logs[path] = md5sum
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
	}
	return logs, nil
}

func putLog(log transfer, bucket *s3.Bucket, dry bool) {
	data, err := ioutil.ReadFile(log.Src)
	if err != nil {
		// Error reading log
		fmt.Printf("Error reading source file %s:\n", log.Src)
		panic(err.Error())
	}

	contType := "binary/octet-stream"
	perm := s3.ACL("private")

	if dry {
		fmt.Printf("Starting sync of %s to bucket path %s...\n", log.Src, log.Dest)
	} else {
		fmt.Printf("Starting sync of %s to s3://%s/%s...\n", log.Src, bucket.Name, log.Dest)
		err = bucket.Put(log.Dest, data, contType, perm, s3.Options{})
		if err != nil {
			// Error uploading log to s3
			fmt.Printf("Sync of %s to s3://%s/%s failed:\n", log.Src, bucket.Name, log.Dest)
			panic(err.Error())
		}
	}
}

func syncFile(log transfer, bucket *s3.Bucket, workerChan chan empty, dry bool) {
	putLog(log, bucket, dry)
	<-workerChan
}

func workerSpawner(bucket *s3.Bucket, fileChan chan transfer, workerChan chan empty, dieChan chan empty, dry bool) {
	for {
		select {
		case file := <-fileChan:
			go syncFile(file, bucket, workerChan, dry)
		case <-dieChan:
			return
		}
	}
}

func (s *SyncT) syncLogs(src, dest map[string]string) error {
	fileChan := make(chan transfer)
	workerChan := make(chan empty, s.Threads)
	dieChan := make(chan empty)
	go workerSpawner(s.Bucket, fileChan, workerChan, dieChan, s.Dry)

	for log, _ := range src {
		if dest[log] != src[log] {
			srcPath := strings.Join([]string{s.Dir, log}, "/")
			destPath := strings.Join([]string{s.Prefix, log}, "/")
			workerChan <- empty{}
			fileChan <- transfer{srcPath, destPath}
		}
	}
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

