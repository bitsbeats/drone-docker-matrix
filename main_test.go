package main

import (
	"sort"
	"strings"
	"testing"
	"os"

	"github.com/google/go-cmp/cmp"
	log "github.com/sirupsen/logrus"
)

func TestBuild(t *testing.T) {
	c = config{
		Registry:         "localhost:5000",
		DefaultNamespace: "images",
		BuildPoolSize:    10,
		UploadPoolSize:   10,
		TagName:          "latest",
		Command:          "echo",
		Workdir:          "testdata",
	}

	os.Setenv("VERSION_FROM_ENV" , "7.3")
	os.Setenv("NAME_FROM_ENV" , "test")

	var got string
	scan(c.Workdir, func(finished chan *build) {
		for b := range finished {
			got += string(b.Output)
			log.Infof("%s Done           %s", b.ID, b.prettyName())
			matrixWg.Done()
		}
	})

	want := `build php -t localhost:5000/images/php:latest-7.2-alpine-test -t docker.io/bitsbeats/image1:latest-7.2-alpine-test -t docker.io/bitsbeats/image2:latest-7.2-alpine-test
push localhost:5000/images/php:latest-7.2-alpine-test
push docker.io/bitsbeats/image1:latest-7.2-alpine-test
push docker.io/bitsbeats/image2:latest-7.2-alpine-test
build php -t localhost:5000/images/php:latest-7.3-alpine-test -t docker.io/bitsbeats/image1:latest-7.3-alpine-test -t docker.io/bitsbeats/image2:latest-7.3-alpine-test
push localhost:5000/images/php:latest-7.3-alpine-test
push docker.io/bitsbeats/image1:latest-7.3-alpine-test
push docker.io/bitsbeats/image2:latest-7.3-alpine-test
build php -t localhost:5000/images/php:latest-7.2-debian-test -t docker.io/bitsbeats/image1:latest-7.2-debian-test -t docker.io/bitsbeats/image2:latest-7.2-debian-test
push localhost:5000/images/php:latest-7.2-debian-test
push docker.io/bitsbeats/image1:latest-7.2-debian-test
push docker.io/bitsbeats/image2:latest-7.2-debian-test
build php -t localhost:5000/images/php:latest-7.3-debian-test -t docker.io/bitsbeats/image1:latest-7.3-debian-test -t docker.io/bitsbeats/image2:latest-7.3-debian-test
push localhost:5000/images/php:latest-7.3-debian-test
push docker.io/bitsbeats/image1:latest-7.3-debian-test
push docker.io/bitsbeats/image2:latest-7.3-debian-test
build busybox -t localhost:5000/images/busybox:latest
push localhost:5000/images/busybox:latest
build python -t localhost:5000/images/python:latest-2.7-alpine
build python -t localhost:5000/images/python:latest-2.7-stretch
build python -t localhost:5000/images/python:latest-3.6-alpine
build python -t localhost:5000/images/python:latest-3.6-stretch
push localhost:5000/images/python:latest-2.7-alpine
push localhost:5000/images/python:latest-2.7-stretch
push localhost:5000/images/python:latest-3.6-alpine
push localhost:5000/images/python:latest-3.6-stretch
`

	wantList := strings.Split(want, "\n")
	gotList := strings.Split(got, "\n")

	sort.Strings(wantList)
	sort.Strings(gotList)

	if diff := cmp.Diff(wantList, gotList); diff != "" {
		t.Errorf("Command mismatch (want, got):\n%s", diff)
	}

}
