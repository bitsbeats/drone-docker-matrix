package main

import (
	"bytes"
	"fmt"
	"net/http"
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
		PushGateway:      "http://vm277.netzmarkt.lan:27121/metrics",
		Time:             time.Now(),
	}

	os.Setenv("VERSION_FROM_ENV", "7.3")
	os.Setenv("NAME_FROM_ENV", "test")
	os.Setenv("DRONE_COMMIT_REF", "279d9035886d4c0427549863c4c2101e4a63e041")
	os.Setenv("DRONE_REPO_LINK", "octocat/matrixed")

	var got string
	b := NewBuilder(
		builder,
		uploader,
		func(b *DockerBuild) {
			got += string(b.Output)
			log.Infof("Done           %s", b.prettyName())

			// notify pushgateway if set
			if c.PushGateway != "" {
				buffer := bytes.NewBuffer([]byte("# TYPE drone_docker_matrix gauge\n"))
				for _, tag := range b.tags() {
					fmt.Fprintf(buffer, "drone_docker_matrix{tag=%q} %d\n", tag, c.Time.Unix())
				}
				url := fmt.Sprintf(
					"%s/job/drone-docker-matrix/image/%s",
					c.PushGateway,
					b.Name,
				)
				_, _ = http.Post(url, "text", bytes.NewReader(buffer.Bytes()))
			}
		},
	)
	err := b.Run(c.Workdir)
	if err != nil {
		t.Fatalf("failed to run: %s", err)
	}

	want := `
build alpine -f alpine/Dockerfile --build-arg MESSAGE=multiply -t localhost:5000/images/alpine:multiply -t localhost:5000/images/alpine:multiply-7
build alpine -f alpine/Dockerfile -t localhost:5000/images/alpine:latest -t localhost:5000/images/alpine:7
build busybox -f busybox/Dockerfile -t localhost:5000/images/busybox:latest -t localhost:5000/images/busybox:7
build https://github.com/openshift/origin-aggregated-logging.git#release-3.11:fluentd -f Dockerfile.centos7 -t localhost:5000/images/remote:latest -t localhost:5000/images/remote:7
build php -f php/Dockerfile --build-arg VERSION=7.2 --build-arg OS=alpine --build-arg NAME=test -t docker.io/bitsbeats/image1:7.2-alpine-test -t docker.io/bitsbeats/image1:7.2-alpine-test-7 -t docker.io/bitsbeats/image2:7.2-alpine-test -t docker.io/bitsbeats/image2:7.2-alpine-test-7 -t localhost:5000/images/php:7.2-alpine-test -t localhost:5000/images/php:7.2-alpine-test-7
build php -f php/Dockerfile --build-arg VERSION=7.2 --build-arg OS=debian --build-arg NAME=test -t docker.io/bitsbeats/image1:7.2-debian-test -t docker.io/bitsbeats/image1:7.2-debian-test-7 -t docker.io/bitsbeats/image2:7.2-debian-test -t docker.io/bitsbeats/image2:7.2-debian-test-7 -t localhost:5000/images/php:7.2-debian-test -t localhost:5000/images/php:7.2-debian-test-7
build php -f php/Dockerfile --build-arg VERSION=7.3 --build-arg OS=alpine --build-arg NAME=test -t docker.io/bitsbeats/image1:7.3-alpine-test -t docker.io/bitsbeats/image1:7.3-alpine-test-7 -t docker.io/bitsbeats/image2:7.3-alpine-test -t docker.io/bitsbeats/image2:7.3-alpine-test-7 -t localhost:5000/images/php:7.3-alpine-test -t localhost:5000/images/php:7.3-alpine-test-7
build php -f php/Dockerfile --build-arg VERSION=7.3 --build-arg OS=debian --build-arg NAME=test -t docker.io/bitsbeats/image1:7.3-debian-test -t docker.io/bitsbeats/image1:7.3-debian-test-7 -t docker.io/bitsbeats/image2:7.3-debian-test -t docker.io/bitsbeats/image2:7.3-debian-test-7 -t localhost:5000/images/php:7.3-debian-test -t localhost:5000/images/php:7.3-debian-test-7
build php -f php/Dockerfile --build-arg VERSION=8.3 --build-arg OS=centos --build-arg NAME=four -t docker.io/bitsbeats/image1:8.3-centos-four -t docker.io/bitsbeats/image1:8.3-centos-four-7 -t docker.io/bitsbeats/image2:8.3-centos-four -t docker.io/bitsbeats/image2:8.3-centos-four-7 -t localhost:5000/images/php:8.3-centos-four -t localhost:5000/images/php:8.3-centos-four-7
build php-custom -f php-custom/Dockerfile --build-arg VERSION=7.2 -t localhost:5000/images/php-custom:7.2 -t localhost:5000/images/php-custom:7.2-7
build php-custom -f php-custom/Dockerfile --build-arg VERSION=7.3 -t localhost:5000/images/php-custom:7.3 -t localhost:5000/images/php-custom:7.3-7
build php-custom -f php-custom/Dockerfile --build-arg VERSION=7.4 --build-arg SPECIAL=green -t localhost:5000/images/php-custom:7.4-green -t localhost:5000/images/php-custom:7.4-green-7
build php-custom -f php-custom/Dockerfile --build-arg VERSION=7.4 -t localhost:5000/images/php-custom:7.4 -t localhost:5000/images/php-custom:7.4-7
build php-custom -f php-custom/Dockerfile --build-arg VERSION=8.0 -t localhost:5000/images/php-custom:8.0 -t localhost:5000/images/php-custom:8.0-7
build php-custom -f php-custom/Dockerfile --build-arg VERSION=8.1 --build-arg SPECIAL=blue -t localhost:5000/images/php-custom:8.1-blue -t localhost:5000/images/php-custom:8.1-blue-7
build php-custom -f php-custom/Dockerfile --build-arg VERSION=8.1 -t localhost:5000/images/php-custom:8.1 -t localhost:5000/images/php-custom:8.1-7
build php-custom -f php-custom/Dockerfile --build-arg VERSION=8.2 --build-arg SPECIAL=blue -t localhost:5000/images/php-custom:8.2-blue -t localhost:5000/images/php-custom:8.2-blue-7
build php-custom -f php-custom/Dockerfile --build-arg VERSION=8.2 -t localhost:5000/images/php-custom:8.2 -t localhost:5000/images/php-custom:8.2-7
build php-custom -f php-custom/Dockerfile --build-arg VERSION=8.3 --build-arg SPECIAL=blue -t localhost:5000/images/php-custom:8.3-blue -t localhost:5000/images/php-custom:8.3-blue-7
build php-custom -f php-custom/Dockerfile --build-arg VERSION=8.3 -t localhost:5000/images/php-custom:8.3 -t localhost:5000/images/php-custom:8.3-7
build php-custom -f php-custom/Dockerfile --build-arg VERSION=8.4 -t localhost:5000/images/php-custom:8.4 -t localhost:5000/images/php-custom:8.4-7
build php-custom -f php-custom/Dockerfile -t localhost:5000/images/php-custom:latest -t localhost:5000/images/php-custom:7
build python -f python/Dockerfile --build-arg VERSION=2.7 --build-arg OS=alpine -t localhost:5000/images/python:2.7-alpine -t localhost:5000/images/python:2.7-alpine-7
build python -f python/Dockerfile --build-arg VERSION=2.7 --build-arg OS=stretch -t localhost:5000/images/python:2.7-stretch -t localhost:5000/images/python:2.7-stretch-7
build python -f python/Dockerfile --build-arg VERSION=3.6 --build-arg OS=alpine -t localhost:5000/images/python:latest -t localhost:5000/images/python:3.6-alpine -t localhost:5000/images/python:3.6-alpine-7
build python -f python/Dockerfile --build-arg VERSION=3.6 --build-arg OS=stretch -t localhost:5000/images/python:3.6-stretch -t localhost:5000/images/python:3.6-stretch-7
build velero -f velero/Dockerfile --build-arg FOR=aws --build-arg VERSION=v1.0.0 -t localhost:5000/images/velero:aws-v1.0.0 -t localhost:5000/images/velero:aws-v1.0.0-7
build velero -f velero/Dockerfile --build-arg FOR=aws --build-arg VERSION=v1.1.0 -t localhost:5000/images/velero:aws-v1.1.0 -t localhost:5000/images/velero:aws-v1.1.0-7
build velero -f velero/Dockerfile --build-arg FOR=gcp --build-arg VERSION=v1.0.0 -t localhost:5000/images/velero:gcp-v1.0.0 -t localhost:5000/images/velero:gcp-v1.0.0-7
push docker.io/bitsbeats/image1:7.2-alpine-test
push docker.io/bitsbeats/image1:7.2-alpine-test-7
push docker.io/bitsbeats/image1:7.2-debian-test
push docker.io/bitsbeats/image1:7.2-debian-test-7
push docker.io/bitsbeats/image1:7.3-alpine-test
push docker.io/bitsbeats/image1:7.3-alpine-test-7
push docker.io/bitsbeats/image1:7.3-debian-test
push docker.io/bitsbeats/image1:7.3-debian-test-7
push docker.io/bitsbeats/image1:8.3-centos-four
push docker.io/bitsbeats/image1:8.3-centos-four-7
push docker.io/bitsbeats/image2:7.2-alpine-test
push docker.io/bitsbeats/image2:7.2-alpine-test-7
push docker.io/bitsbeats/image2:7.2-debian-test
push docker.io/bitsbeats/image2:7.2-debian-test-7
push docker.io/bitsbeats/image2:7.3-alpine-test
push docker.io/bitsbeats/image2:7.3-alpine-test-7
push docker.io/bitsbeats/image2:7.3-debian-test
push docker.io/bitsbeats/image2:7.3-debian-test-7
push docker.io/bitsbeats/image2:8.3-centos-four
push docker.io/bitsbeats/image2:8.3-centos-four-7
push localhost:5000/images/alpine:7
push localhost:5000/images/alpine:latest
push localhost:5000/images/alpine:multiply
push localhost:5000/images/alpine:multiply-7
push localhost:5000/images/busybox:7
push localhost:5000/images/busybox:latest
push localhost:5000/images/php-custom:7
push localhost:5000/images/php-custom:7.2
push localhost:5000/images/php-custom:7.2-7
push localhost:5000/images/php-custom:7.3
push localhost:5000/images/php-custom:7.3-7
push localhost:5000/images/php-custom:7.4
push localhost:5000/images/php-custom:7.4-7
push localhost:5000/images/php-custom:7.4-green
push localhost:5000/images/php-custom:7.4-green-7
push localhost:5000/images/php-custom:8.0
push localhost:5000/images/php-custom:8.0-7
push localhost:5000/images/php-custom:8.1
push localhost:5000/images/php-custom:8.1-7
push localhost:5000/images/php-custom:8.1-blue
push localhost:5000/images/php-custom:8.1-blue-7
push localhost:5000/images/php-custom:8.2
push localhost:5000/images/php-custom:8.2-7
push localhost:5000/images/php-custom:8.2-blue
push localhost:5000/images/php-custom:8.2-blue-7
push localhost:5000/images/php-custom:8.3
push localhost:5000/images/php-custom:8.3-7
push localhost:5000/images/php-custom:8.3-blue
push localhost:5000/images/php-custom:8.3-blue-7
push localhost:5000/images/php-custom:8.4
push localhost:5000/images/php-custom:8.4-7
push localhost:5000/images/php-custom:latest
push localhost:5000/images/php:7.2-alpine-test
push localhost:5000/images/php:7.2-alpine-test-7
push localhost:5000/images/php:7.2-debian-test
push localhost:5000/images/php:7.2-debian-test-7
push localhost:5000/images/php:7.3-alpine-test
push localhost:5000/images/php:7.3-alpine-test-7
push localhost:5000/images/php:7.3-debian-test
push localhost:5000/images/php:7.3-debian-test-7
push localhost:5000/images/php:8.3-centos-four
push localhost:5000/images/php:8.3-centos-four-7
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
push localhost:5000/images/velero:aws-v1.0.0
push localhost:5000/images/velero:aws-v1.0.0-7
push localhost:5000/images/velero:aws-v1.1.0
push localhost:5000/images/velero:aws-v1.1.0-7
push localhost:5000/images/velero:gcp-v1.0.0
push localhost:5000/images/velero:gcp-v1.0.0-7
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

	fa, err := os.Create("/tmp/a")
	if err != nil {
		log.Fatalf("error: %s", err)
	}
	defer fa.Close()
	fb, err := os.Create("/tmp/b")
	if err != nil {
		log.Fatalf("error: %s", err)
	}
	defer fb.Close()
	for _, str := range wantList {
		_, err = fmt.Fprintln(fa, str)
		if err != nil {
			log.Fatalf("error: %s", err)
		}

	}
	for _, str := range gotList {
		fmt.Fprintln(fb, str)
	}

	if diff := cmp.Diff(wantList, gotList); diff != "" {
		t.Errorf("Command mismatch (want, got):\n%s", diff)
	}

}
