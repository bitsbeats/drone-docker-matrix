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
		output chan<- *Build
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

	// without docker-matrix.yaml its just a normal build
	_, err := os.Stat(matrixFile)
	if os.IsNotExist(err) {
		tag := c.TagName
		if tag == "" {
			tag = "latest"
		}
		dockerfile := filepath.Join(path, "Dockerfile")
		p.wg.Add(1)
		defer p.wg.Done()
		b := Build{
			ID:              id,
			Namespace:       c.DefaultNamespace,
			Name:            name,
			Path:            path,
			Dockerfile:      dockerfile,
			Tag:             tag,
			Scenario:        make(map[string]string),
			AdditionalNames: []string{},
		}
		p.output <- &b
		return nil
	} else if err != nil {
		return fmt.Errorf("unable to stat matrixfile: %w", err)
	}

	// load matrix
	fileContent, err := ioutil.ReadFile(matrixFile)
	if err != nil {
		return fmt.Errorf("%s failed to load %s", id, matrixFile)
	}
	var m matrix
	err = yaml.Unmarshal(fileContent, &m)
	if err != nil {
		return fmt.Errorf("%s unable to parse '%s': %w", id, matrixFile, err)
	}
	if m.CustomPath != "" {
		path = m.CustomPath
	}

	// add directory to dockerfiles
	if m.CustomDockerfile == "" {
		m.CustomDockerfile = "Dockerfile"
	}
	if !strings.Contains(path, "://") {
		m.CustomDockerfile = filepath.Join(path, m.CustomDockerfile)
	}

	// multiply options
	keyOrder := []string{}
	scenariosMatrix := []map[string]string{map[string]string{}}
MATRIX:
	for _, multiplyItem := range m.Multiply {
		// type conversion (no matter what the yaml has, we want strings)
		argument := fmt.Sprintf("%v", multiplyItem.Key)
		values := []string{}
		for _, value := range multiplyItem.Value.([]interface{}) {
			stringValue := fmt.Sprintf("%v", value)
			parsedValue, err := envsubst.EvalEnv(stringValue)
			if err != nil {
				log.Errorf("%s unable to envsubst %s -> %s: %s", id, argument, stringValue, err)
				continue MATRIX
			}
			values = append(values, parsedValue)
		}

		// iterate over options and build matrix
		keyOrder = append(keyOrder, argument)
		var scenariosNew []map[string]string
		for _, value := range values {
			for _, scenario := range scenariosMatrix {
				scenarioNew := make(map[string]string)
				for k, v := range scenario {
					scenarioNew[k] = v
				}
				scenarioNew[argument] = value
				scenariosNew = append(scenariosNew, scenarioNew)
			}
		}
		scenariosMatrix = scenariosNew
	}

	// append options
ARGUMENTS:
	for _, scenarioMatrix := range scenariosMatrix {
		scenario := make(map[string]string)
		for k, v := range scenarioMatrix {
			scenario[k] = v
		}
		keyOrder := append(keyOrder[:0:0], keyOrder...)
		for _, apnd := range m.Append {
			for k, v := range apnd {
				vParsed, err := envsubst.EvalEnv(v)
				if err != nil {
					log.Errorf("%s unable to envsubst %s: %s", id, v, err)
					continue ARGUMENTS
				}
				scenario[k] = vParsed
				keyOrder = append(keyOrder, k)
			}
		}

		// generate tag
		tag := c.TagName
		if tag == "" {
			tag = "latest"
		}
		for _, key := range keyOrder {
			t := scenario[key]
			if t == "" {
				log.Debugf("%s skipping empty tag in name for %s", id, key)
				continue
			}
			tag = fmt.Sprintf("%s-%s", tag, t)
		}
		if tag[0:1] == "-" {
			tag = tag[1:]
		}

		// RUN
		namespace := c.DefaultNamespace
		if m.Namespace != "" {
			namespace = m.Namespace
		}
		p.wg.Add(1)
		defer p.wg.Done()
		b := Build{
			ID:              id,
			Namespace:       namespace,
			Name:            name,
			Path:            path,
			Tag:             tag,
			Scenario:        scenario,
			AdditionalNames: m.AdditionalNames,
			KeyOrder:        keyOrder,
			AsLatest:        m.AsLatest,
			Dockerfile:      m.CustomDockerfile,
		}
		p.output <- &b
	}
	return nil
}
