package main

import (
	"os"
	"sort"
	"strings"
	"testing"

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
		TagBuildID:       "7",
		Command:          "echo",
		Workdir:          "testdata",
	}

	os.Setenv("VERSION_FROM_ENV", "7.3")
	os.Setenv("NAME_FROM_ENV", "test")

	var got string
	scan(c.Workdir, func(finished chan *build) {
		for b := range finished {
			got += string(b.Output)
			log.Infof("%s Done           %s", b.ID, b.prettyName())
			matrixWg.Done()
		}
	})

	want := `build busybox -t localhost:5000/images/busybox:latest -t localhost:5000/images/busybox:7
build python -t localhost:5000/images/python:2.7-alpine -t localhost:5000/images/python:2.7-alpine-7
build python -t localhost:5000/images/python:2.7-stretch -t localhost:5000/images/python:2.7-stretch-7
build python -t localhost:5000/images/python:3.6-alpine -t localhost:5000/images/python:3.6-alpine-7
build python -t localhost:5000/images/python:3.6-stretch -t localhost:5000/images/python:3.6-stretch-7
build php -t docker.io/bitsbeats/image1:7.2-alpine-test -t docker.io/bitsbeats/image1:7.2-alpine-test-7 -t docker.io/bitsbeats/image2:7.2-alpine-test -t docker.io/bitsbeats/image2:7.2-alpine-test-7 -t localhost:5000/images/php:7.2-alpine-test -t localhost:5000/images/php:7.2-alpine-test-7
build php -t docker.io/bitsbeats/image1:7.2-debian-test -t docker.io/bitsbeats/image1:7.2-debian-test-7 -t docker.io/bitsbeats/image2:7.2-debian-test -t docker.io/bitsbeats/image2:7.2-debian-test-7 -t localhost:5000/images/php:7.2-debian-test -t localhost:5000/images/php:7.2-debian-test-7
build php -t docker.io/bitsbeats/image1:7.3-alpine-test -t docker.io/bitsbeats/image1:7.3-alpine-test-7 -t docker.io/bitsbeats/image2:7.3-alpine-test -t docker.io/bitsbeats/image2:7.3-alpine-test-7 -t localhost:5000/images/php:7.3-alpine-test -t localhost:5000/images/php:7.3-alpine-test-7
build php -t docker.io/bitsbeats/image1:7.3-debian-test -t docker.io/bitsbeats/image1:7.3-debian-test-7 -t docker.io/bitsbeats/image2:7.3-debian-test -t docker.io/bitsbeats/image2:7.3-debian-test-7 -t localhost:5000/images/php:7.3-debian-test -t localhost:5000/images/php:7.3-debian-test-7
push docker.io/bitsbeats/image1:7.2-alpine-test
push docker.io/bitsbeats/image1:7.2-alpine-test-7
push docker.io/bitsbeats/image1:7.2-debian-test
push docker.io/bitsbeats/image1:7.2-debian-test-7
push docker.io/bitsbeats/image1:7.3-alpine-test
push docker.io/bitsbeats/image1:7.3-alpine-test-7
push docker.io/bitsbeats/image1:7.3-debian-test
push docker.io/bitsbeats/image1:7.3-debian-test-7
push docker.io/bitsbeats/image2:7.2-alpine-test
push docker.io/bitsbeats/image2:7.2-alpine-test-7
push docker.io/bitsbeats/image2:7.2-debian-test
push docker.io/bitsbeats/image2:7.2-debian-test-7
push docker.io/bitsbeats/image2:7.3-alpine-test
push docker.io/bitsbeats/image2:7.3-alpine-test-7
push docker.io/bitsbeats/image2:7.3-debian-test
push docker.io/bitsbeats/image2:7.3-debian-test-7
push localhost:5000/images/busybox:latest
push localhost:5000/images/busybox:7
push localhost:5000/images/php:7.2-alpine-test
push localhost:5000/images/php:7.2-alpine-test-7
push localhost:5000/images/php:7.2-debian-test
push localhost:5000/images/php:7.2-debian-test-7
push localhost:5000/images/php:7.3-alpine-test
push localhost:5000/images/php:7.3-alpine-test-7
push localhost:5000/images/php:7.3-debian-test
push localhost:5000/images/php:7.3-debian-test-7
push localhost:5000/images/python:2.7-alpine
push localhost:5000/images/python:2.7-alpine-7
push localhost:5000/images/python:2.7-stretch
push localhost:5000/images/python:2.7-stretch-7
push localhost:5000/images/python:3.6-alpine
push localhost:5000/images/python:3.6-alpine-7
push localhost:5000/images/python:3.6-stretch
push localhost:5000/images/python:3.6-stretch-7
`

	wantList := strings.Split(want, "\n")
	gotList := strings.Split(got, "\n")

	sort.Strings(wantList)
	sort.Strings(gotList)

	if diff := cmp.Diff(wantList, gotList); diff != "" {
		t.Errorf("Command mismatch (want, got):\n%s", diff)
	}

}
