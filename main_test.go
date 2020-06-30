package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

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
		Time:             time.Now(),
	}

	os.Setenv("VERSION_FROM_ENV", "7.3")
	os.Setenv("NAME_FROM_ENV", "test")
	os.Setenv("DRONE_COMMIT_REF", "279d9035886d4c0427549863c4c2101e4a63e041")
	os.Setenv("DRONE_REPO_LINK", "octocat/matrixed")

	var got string
	scan(c.Workdir, func(finished chan *build) {
		for b := range finished {
			got += string(b.Output)
			log.Infof("%s Done           %s", b.ID, b.prettyName())
			matrixWg.Done()
		}
	})

	want := `
build alpine -f alpine/Dockerfile --build-arg MESSAGE=multiply -t localhost:5000/images/alpine:multiply -t localhost:5000/images/alpine:multiply-7
build alpine -f alpine/Dockerfile -t localhost:5000/images/alpine:latest -t localhost:5000/images/alpine:7
build busybox -f busybox/Dockerfile -t localhost:5000/images/busybox:latest -t localhost:5000/images/busybox:7
build https://github.com/openshift/origin-aggregated-logging.git#release-3.11:fluentd -f Dockerfile.centos7 -t localhost:5000/images/remote:latest -t localhost:5000/images/remote:7
build php -f php/Dockerfile --build-arg VERSION=7.2 --build-arg OS=alpine --build-arg NAME=test -t docker.io/bitsbeats/image1:7.2-alpine-test -t docker.io/bitsbeats/image1:7.2-alpine-test-7 -t docker.io/bitsbeats/image2:7.2-alpine-test -t docker.io/bitsbeats/image2:7.2-alpine-test-7 -t localhost:5000/images/php:7.2-alpine-test -t localhost:5000/images/php:7.2-alpine-test-7
build php -f php/Dockerfile --build-arg VERSION=7.2 --build-arg OS=debian --build-arg NAME=test -t docker.io/bitsbeats/image1:7.2-debian-test -t docker.io/bitsbeats/image1:7.2-debian-test-7 -t docker.io/bitsbeats/image2:7.2-debian-test -t docker.io/bitsbeats/image2:7.2-debian-test-7 -t localhost:5000/images/php:7.2-debian-test -t localhost:5000/images/php:7.2-debian-test-7
build php -f php/Dockerfile --build-arg VERSION=7.3 --build-arg OS=alpine --build-arg NAME=test -t docker.io/bitsbeats/image1:7.3-alpine-test -t docker.io/bitsbeats/image1:7.3-alpine-test-7 -t docker.io/bitsbeats/image2:7.3-alpine-test -t docker.io/bitsbeats/image2:7.3-alpine-test-7 -t localhost:5000/images/php:7.3-alpine-test -t localhost:5000/images/php:7.3-alpine-test-7
build php -f php/Dockerfile --build-arg VERSION=7.3 --build-arg OS=debian --build-arg NAME=test -t docker.io/bitsbeats/image1:7.3-debian-test -t docker.io/bitsbeats/image1:7.3-debian-test-7 -t docker.io/bitsbeats/image2:7.3-debian-test -t docker.io/bitsbeats/image2:7.3-debian-test-7 -t localhost:5000/images/php:7.3-debian-test -t localhost:5000/images/php:7.3-debian-test-7
build python -f python/Dockerfile --build-arg VERSION=2.7 --build-arg OS=alpine -t localhost:5000/images/python:2.7-alpine -t localhost:5000/images/python:2.7-alpine-7
build python -f python/Dockerfile --build-arg VERSION=2.7 --build-arg OS=stretch -t localhost:5000/images/python:2.7-stretch -t localhost:5000/images/python:2.7-stretch-7
build python -f python/Dockerfile --build-arg VERSION=3.6 --build-arg OS=alpine -t localhost:5000/images/python:latest -t localhost:5000/images/python:3.6-alpine -t localhost:5000/images/python:3.6-alpine-7
build python -f python/Dockerfile --build-arg VERSION=3.6 --build-arg OS=stretch -t localhost:5000/images/python:3.6-stretch -t localhost:5000/images/python:3.6-stretch-7
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
push localhost:5000/images/alpine:7
push localhost:5000/images/alpine:latest
push localhost:5000/images/alpine:multiply
push localhost:5000/images/alpine:multiply-7
push localhost:5000/images/busybox:7
push localhost:5000/images/busybox:latest
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
push localhost:5000/images/python:latest
push localhost:5000/images/remote:7
push localhost:5000/images/remote:latest
`
	want = want[1:]

	wantList := strings.Split(want, "\n")
	for i, item := range wantList {
		if strings.HasPrefix(item, "build ") {
			wantList[i] = item +
				" --label org.label-schema.schema-version=1.0" +
				" --label org.label-schema.vcs-ref=279d9035886d4c0427549863c4c2101e4a63e041" +
				" --label org.label-schema.vcs-url=octocat/matrixed" +
				fmt.Sprintf(" --label org.label-schema.build-date=%s", c.Time.Format(time.RFC3339))
		}
	}
	gotList := strings.Split(got, "\n")

	sort.Strings(wantList)
	sort.Strings(gotList)

	if diff := cmp.Diff(wantList, gotList); diff != "" {
		t.Errorf("Command mismatch (want, got):\n%s", diff)
	}

}
