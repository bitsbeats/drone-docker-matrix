package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/segmentio/ksuid"
	log "github.com/sirupsen/logrus"
)

type (
	// DockerBuild stores the build infomration
	DockerBuild struct {
		ID        ksuid.KSUID
		Namespace string
		Name      string

		Path       string
		Dockerfile string
		Tag        string

		Arguments     map[string]string
		ArgumentOrder []string

		AdditionalNames []string
		Output          []byte
		AsLatest        string

		// Froms stores all Dockerfile `FROM` commands
		Froms []string

		Error error
	}
)

func NewDockerBuild(id ksuid.KSUID, name, path string) *DockerBuild {
	return &DockerBuild{
		ID:   id,
		Name: name,
		Path: path,
	}
}

// copy create a deep copy of the Dockerbuild
func (b *DockerBuild) copy() *DockerBuild {
	arguments := make(map[string]string, len(b.Arguments))
	for key, val := range b.Arguments {
		arguments[key] = val
	}
	return &DockerBuild{
		ID:              b.ID,
		Namespace:       b.Namespace,
		Name:            b.Name,
		Path:            b.Path,
		Dockerfile:      b.Dockerfile,
		Tag:             b.Tag,
		Arguments:       arguments,
		ArgumentOrder:   append(b.ArgumentOrder[0:0], b.ArgumentOrder...),
		AdditionalNames: append(b.AdditionalNames[0:0], b.AdditionalNames...),
		Output:          append(b.Output[0:0], b.Output...),
		AsLatest:        b.AsLatest,
		Froms:           append(b.Froms[0:0], b.Froms...),
		Error:           b.Error,
	}
}

// copyWithArgument create a copy with a new build argument added
func (b *DockerBuild) copyWithArgument(name, value string) *DockerBuild {
	result := b.copy()
	result.ArgumentOrder = append(b.ArgumentOrder, name)
	result.Arguments[name] = value
	if b.Tag == "" {
		result.Tag = value
	} else if value != "" {
		result.Tag = fmt.Sprintf("%s-%s", b.Tag, value)
	}
	return result
}

// prettyName
func (b *DockerBuild) prettyName() string {
	tag := strings.TrimPrefix(b.Tag, "latest-")
	return fmt.Sprintf("%s:%s", b.Name, tag)
}

// gather tags
func (b *DockerBuild) tags() (combined []string) {
	images := append(b.AdditionalNames, fmt.Sprintf("%s/%s/%s", c.Registry, b.Namespace, b.Name))

	tags := []string{b.Tag}
	if c.TagBuildID != "" {
		tags = append(
			tags,
			strings.TrimPrefix(fmt.Sprintf("%s-%s", b.Tag, c.TagBuildID), "-"),
		)
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
func (b *DockerBuild) args() []string {
	log.Warnf("Building       %s from %s dockerfile:%s", b.prettyName(), b.Path, b.Dockerfile)
	args := []string{"build", b.Path}
	if b.Dockerfile != "" {
		args = append(args, "-f", b.Dockerfile)
	}
	for _, k := range b.ArgumentOrder {
		if b.Arguments[k] == "" {
			log.Infof("skipping empty build arg %s", k)
			continue
		}
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", k, b.Arguments[k]))
	}
	for _, tag := range b.tags() {
		args = append(args, "-t", tag)
	}
	if c.Pull {
		args = append(args, "--pull")
	}

	args = append(args,
		"--label", fmt.Sprintf("org.label-schema.schema-version=%s", "1.0"),
		"--label", fmt.Sprintf("org.label-schema.vcs-ref=%s", os.Getenv("DRONE_COMMIT_REF")),
		"--label", fmt.Sprintf("org.label-schema.vcs-url=%s", os.Getenv("DRONE_REPO_LINK")),
		"--label", fmt.Sprintf("org.label-schema.build-date=%s", c.Time.Format(time.RFC3339)),
	)

	return args
}

// build builds the image
func (b *DockerBuild) build() (err error) {
	cmd := exec.Command(c.Command, b.args()...)
	_ = cmd.Wait()
	b.Output, err = cmd.CombinedOutput()
	return err
}

// upload uploads the image
func (b *DockerBuild) upload() (err error) {
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

// parseFromsFromDockerfile searches for all FROM statements and builds a list
// of all FROM images
func parseFromsFromDockerfile(path string) ([]string, error) {
	dockerfile, err := ioutil.ReadFile(path)
	if err != nil {
		return []string{}, fmt.Errorf("unable to read dockerfile: %w", err)
	}
	fromCommand := regexp.MustCompile(`^FROM +([^ ]+)`)
	matches := fromCommand.FindAllStringSubmatch(string(dockerfile), -1)
	froms := make([]string, len(matches))
	for i, match := range matches {
		froms[i] = match[1]
	}
	return froms, nil
}
