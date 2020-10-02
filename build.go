package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/segmentio/ksuid"
	log "github.com/sirupsen/logrus"
)

type (
	// Build stores the build infomration
	Build struct {
		ID        ksuid.KSUID
		Namespace string
		Name      string

		Path            string
		Dockerfile      string
		Tag             string
		Scenario        map[string]string
		KeyOrder        []string
		AdditionalNames []string
		Output          []byte
		AsLatest        string

		Error error
	}
)

// prettyName
func (b *Build) prettyName() string {
	tag := strings.TrimPrefix(b.Tag, "latest-")
	return fmt.Sprintf("%s:%s", b.Name, tag)
}

// gather tags
func (b *Build) tags() (combined []string) {
	images := append(b.AdditionalNames, fmt.Sprintf("%s/%s/%s", c.Registry, b.Namespace, b.Name))

	tags := []string{b.Tag}
	if c.TagBuildID != "" {
		tags = append(tags, fmt.Sprintf("%s-%s", b.Tag, c.TagBuildID))
	}
	for _, name := range images {
		for _, tag := range tags {
			tag = strings.TrimPrefix(tag, "latest-")
			if tag == b.AsLatest {
				combined = append(combined, fmt.Sprintf("%s/%s/%s:latest", c.Registry, b.Namespace, b.Name))
			}
			combined = append(combined, fmt.Sprintf("%s:%s", name, tag))
		}
	}
	return combined
}

// args builds the argument for the docker build
func (b *Build) args() []string {
	log.Warnf("Building       %s from %s dockerfile:%s", b.prettyName(), b.Path, b.Dockerfile)
	args := []string{"build", b.Path}
	if b.Dockerfile != "" {
		args = append(args, "-f", b.Dockerfile)
	}
	for _, k := range b.KeyOrder {
		if b.Scenario[k] == "" {
			log.Infof("skipping empty build arg %s", k)
			continue
		}
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", k, b.Scenario[k]))
	}
	for _, tag := range b.tags() {
		args = append(args, "-t", tag)
	}
	if c.Pull {
		args = append(args, "--pull")
	}

	args = append(args,
		"--label", fmt.Sprintf("org.label-schema.schema-version=1.0"),
		"--label", fmt.Sprintf("org.label-schema.vcs-ref=%s", os.Getenv("DRONE_COMMIT_REF")),
		"--label", fmt.Sprintf("org.label-schema.vcs-url=%s", os.Getenv("DRONE_REPO_LINK")),
		"--label", fmt.Sprintf("org.label-schema.build-date=%s", c.Time.Format(time.RFC3339)),
	)

	return args
}

// build builds the image
func (b *Build) build() (err error) {
	cmd := exec.Command(c.Command, b.args()...)
	_ = cmd.Wait()
	b.Output, err = cmd.CombinedOutput()
	return err
}

// upload uploads the image
func (b *Build) upload() (err error) {
	for _, tag := range b.tags() {
		log.Warnf("Uploading      %s", tag)
		cmd := exec.Command(c.Command, "push", tag)
		_ = cmd.Wait()
		subOut, err := cmd.CombinedOutput()
		b.Output = append(b.Output, subOut...)
		if err != nil {
			return err
		}
	}
	return err
}
