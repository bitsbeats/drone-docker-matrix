package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/drone/envsubst"
	"github.com/segmentio/ksuid"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type (
	Parser struct {
		wg     *sync.WaitGroup
		output chan<- *DockerBuild
	}
)

func (p *Parser) WaitAndClose() {
	p.wg.Wait()
	close(p.output)
}

// Parse loads a docker-matrix and creates builds to input
func (p *Parser) Parse(name string) error {
	p.wg.Add(1)
	defer p.wg.Done()

	id := ksuid.New()
	matrixFile := filepath.Join(name, "docker-matrix.yml")

	path := name
	if name == "." {
		p, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("%s unable to get directory name.", id)
		}
		name = filepath.Base(p)
	}

	b := NewDockerBuild(id, name, path)

	// without docker-matrix.yaml its just a normal build
	_, err := os.Stat(matrixFile)
	if os.IsNotExist(err) {
		return p.normalBuild(b)
	} else if err != nil {
		return fmt.Errorf("unable to stat matrixfile: %w", err)
	}

	// otherwise run matrix build
	return p.matrixBuild(b, matrixFile)

}

func (p *Parser) normalBuild(b *DockerBuild) error {
	p.wg.Add(1)
	defer p.wg.Done()

	tag := c.TagName
	if tag == "" {
		tag = "latest"
	}

	b.Namespace = c.DefaultNamespace
	b.Dockerfile = filepath.Join(b.Path, "Dockerfile")
	b.Tag = tag
	b.Arguments = make(map[string]string)
	b.AdditionalNames = []string{}

	p.output <- b
	return nil
}

func (p *Parser) matrixBuild(b *DockerBuild, matrixFile string) error {
	var m Matrix
	err := loadMatrix(matrixFile, b, &m)
	if err != nil {
		return fmt.Errorf("unable to load matrix file")
	}

	// apply settings
	if m.CustomPath != "" {
		b.Path = m.CustomPath
	}
	if m.CustomDockerfile == "" {
		m.CustomDockerfile = "Dockerfile"
	}
	if !strings.Contains(b.Path, "://") {
		m.CustomDockerfile = filepath.Join(b.Path, m.CustomDockerfile)
	}
	namespace := c.DefaultNamespace
	if m.Namespace != "" {
		namespace = m.Namespace
	}

	// if possible add images to cleanup
	froms, err := parseFromsFromDockerfile(m.CustomDockerfile)
	if err != nil {
		log.Warnf("%s unable to parse FROMs in %q: %s", b.ID, m.CustomDockerfile, err)
	}

	// create single build list as base for multiply
	multiplies := m.getEnvSubstedMultiply(b.ID)
	builds := []*DockerBuild{{
		ID:        b.ID,
		Namespace: namespace,
		Name:      b.Name,
		Path:      b.Path,
		Tag:       "",

		Arguments:     map[string]string{},
		ArgumentOrder: []string{},

		AdditionalNames: m.AdditionalNames,
		AsLatest:        m.AsLatest,
		Dockerfile:      m.CustomDockerfile,
		Froms:           froms,
	}}

	// handle multiply arguments
	for _, multiplyItem := range multiplies {
		builds = handleMultiply(builds, multiplyItem.Name, multiplyItem.Values)
	}

	// handle apply arguments
	newBuilds := []*DockerBuild{}
	if len(m.Append) > 0 {
		for _, argValues := range m.Append {
			newBuilds = append(newBuilds, handleAppend(builds, argValues)...)
		}
		builds = newBuilds
	}

	// schedule building
	for _, build := range builds {
		if build.Tag == "" {
			build.Tag = "latest"
		}
		p.wg.Add(1)
		defer p.wg.Done()
		p.output <- build
	}

	return nil
}

func loadMatrix(file string, b *DockerBuild, m *Matrix) error {
	fileContent, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("%s failed to load %s", b.ID, file)
	}
	err = yaml.Unmarshal(fileContent, m)
	if err != nil {
		return fmt.Errorf("%s unable to parse '%s': %w", b.ID, file, err)
	}
	return nil
}

func handleMultiply(builds []*DockerBuild, argName string, argValues []string) []*DockerBuild {
	multiplied := []*DockerBuild{}
	for _, b := range builds {
		for _, argValue := range argValues {
			substed, err := envsubst.EvalEnv(argValue)
			if err == nil {
				argValue = substed
			} else {
				log.Errorf("%s unable to envsubst %s: %s", b.ID, argValue, err)
			}
			multiplied = append(
				multiplied,
				b.copyWithArgument(argName, argValue),
			)
		}
	}
	return multiplied
}

func handleAppend(builds []*DockerBuild, arguments map[string]string) []*DockerBuild {
	appended := []*DockerBuild{}
	for _, build := range builds {
		for argName, argValue := range arguments {
			substed, err := envsubst.EvalEnv(argValue)
			if err == nil {
				argValue = substed
			} else {
				log.Errorf("%s unable to envsubst %s: %s", build.ID, argValue, err)
			}
			build = build.copyWithArgument(argName, argValue)
		}
		appended = append(appended, build)
	}
	return appended
}
