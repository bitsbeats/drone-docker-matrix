package main

import (
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
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

	var got string
	scan(c.Workdir, func(finished chan *build) {
		for b := range finished {
			got += string(b.Output)
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
`

	wantList := strings.Split(want, "\n")
	gotList := strings.Split(got, "\n")

	sort.Strings(wantList)
	sort.Strings(gotList)

	if diff := cmp.Diff(wantList, gotList); diff != "" {
		t.Errorf("Command mismatch (want, got):\n%s", diff)
	}

}
