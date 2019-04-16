package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kelseyhightower/envconfig"
	"github.com/segmentio/ksuid"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type (
	// Configuration
	config struct {
		Registry         string `envconfig:"REGISTRY"`
		DefaultNamespace string `envconfig:"DEFAULT_NAMESPACE" default:"images"`
		BuildPoolSize    int    `envconfig:"BUILD_POOL_SIZE" default:"4"`
		UploadPoolSize   int    `envconfig:"UPLOAD_POOL_SIZE" default:"4"`
		TagName          string `envconfig:"TAG_NAME" default:"latest"`
		Command          string `default:"docker"`
		Workdir          string `default:"."`
	}

	// Matrix that describes all arguments required to build the dockerfile.
	matrix struct {
		// Multiply arguments that are multiplied with each other:
		//
		//   multiply:
		//     VERSION:
		//       - 7.2
		//       - 7.3
		//     OS:
		//       - alpine
		//       - debian
		//
		// will build to:
		//
		//  { VERSION: 7.2, OS: alpine }
		//  { VERSION: 7.3, OS: alpine }
		//  { VERSION: 7.2, OS: debian }
		//  { VERSION: 7.3, OS: debian }
		Multiply yaml.MapSlice `yaml:"multiply"`

		// Append that are added as they are, keys are use for the image tag:
		//
		//   append:
		//     - { PATH: /var/www, DIR: html }
		Append []map[string]string `yaml:"append"`

		// Namespace: namespace for the image
		Namespace string `yaml:"namespace"`

		// AdditionalNames is a comma seperated list of tag names to upload as
		//
		//   additional_names: bitsbeats/drone-docker-matrix
		AdditionalNames []string `yaml:"additional_names"`
	}

	// Build information
	build struct {
		ID              ksuid.KSUID
		Namespace       string
		Name            string
		Tag             string
		Scenario        map[string]string
		KeyOrder        []string
		AdditionalNames []string
		Output          []byte
	}
)

var (
	c        config
	matrixWg sync.WaitGroup
	buildWg  sync.WaitGroup
	uploadWg sync.WaitGroup
)

func main() {
	// configuration
	err := envconfig.Process("plugin", &c)
	if err != nil {
		log.Fatal(err.Error())
	}
	if c.BuildPoolSize < 1 || c.UploadPoolSize < 1 {
		log.Fatalf("PoolSize may not be smaller than 1: BuildPoolSize: %d, UploadPoolSize: %d", c.BuildPoolSize, c.UploadPoolSize)
	}
	log.Infof("Configuration: %+v", c)

	// run
	scan(c.Workdir, func(finished chan *build) {
		for b := range finished {
			log.Infof("%s Done           %s", b.ID, b.prettyName())
			matrixWg.Done()
		}
	})
}

func scan(path string, finisher func(chan *build)) {
	// spawn build and upload pool
	builds := make(chan *build, 128)
	uploads := make(chan *build, 128)
	finished := make(chan *build, 128)
	go finisher(finished)
	go pool(c.BuildPoolSize, builds, uploads, &buildWg, builder)
	go pool(c.UploadPoolSize, uploads, finished, &uploadWg, uploader)

	// check for files
	err := os.Chdir(path)
	if err != nil {
		log.Fatalf("Failed to change directory to %s", path)
	}
	err = filepath.Walk(".", func(file string, info os.FileInfo, err error) error {
		name := filepath.Base(filepath.Dir(file))
		filename := filepath.Base(file)
		if err == nil && filename == "Dockerfile" {
			handleMatrix(name, builds)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err.Error())
	}

	// wait for tasks to finish
	matrixWg.Wait()
	buildWg.Wait()
	uploadWg.Wait()
	close(builds)
	close(uploads)
	close(finished)
}

func handleMatrix(name string, builds chan *build) {
	matrixFile := filepath.Join(name, "docker-matrix.yml")

	// without docker-matrix.yaml its just a normal build
	if _, err := os.Stat(matrixFile); err != nil {
		tag := c.TagName
		if len(tag) == 0 {
			tag = "latest"
		}
		matrixWg.Add(1)
		b := build{
			ID:              ksuid.New(),
			Namespace:       c.DefaultNamespace,
			Name:            name,
			Tag:             tag,
			Scenario:        make(map[string]string),
			AdditionalNames: []string{},
		}
		builds <- &b
		return
	}

	// load matrix
	fileContent, err := ioutil.ReadFile(matrixFile)
	if err != nil {
		log.Errorf("--------------------------- Failed to load %s", matrixFile)
		return
	}
	var m matrix
	err = yaml.Unmarshal(fileContent, &m)
	if err != nil {
		log.Errorf("--------------------------- Unable to parse '%s': %s", matrixFile, err)
		return
	}

	// multiply options
	keyOrder := []string{}
	scenariosMatrix := []map[string]string{map[string]string{}}
	for _, multiplyItem := range m.Multiply {
		// type conversion (no matter what the yaml has, we want strings)
		argument := fmt.Sprintf("%v", multiplyItem.Key)
		values := []string{}
		for _, value := range multiplyItem.Value.([]interface{}) {
			values = append(values, fmt.Sprintf("%v", value))
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
	for _, scenarioMatrix := range scenariosMatrix {
		scenario := make(map[string]string)
		for k, v := range scenarioMatrix {
			scenario[k] = v
		}
		keyOrder := append(keyOrder[:0:0], keyOrder...)
		for _, apnd := range m.Append {
			for k, v := range apnd {
				scenario[k] = v
				keyOrder = append(keyOrder, k)
			}
		}

		// generate tag
		tag := c.TagName
		for _, key := range keyOrder {
			tag = fmt.Sprintf("%s-%s", tag, scenario[key])
		}
		if len(tag) == 0 {
			tag = "latest"
		}
		if tag[0:1] == "-" {
			tag = tag[1:]
		}

		// RUN
		namespace := c.DefaultNamespace
		if m.Namespace != "" {
			namespace = m.Namespace
		}
		matrixWg.Add(1)
		b := build{
			ID:              ksuid.New(),
			Namespace:       namespace,
			Name:            name,
			Tag:             tag,
			Scenario:        scenario,
			AdditionalNames: m.AdditionalNames,
		}
		builds <- &b
	}
}

// pool for paralell builds
func pool(size int, builds chan *build, callback chan *build, wg *sync.WaitGroup, handler func(*build)) {
	p := make(chan bool, size)
	for i := 0; i < size; i++ {
		p <- true
	}

	for b := range builds {
		wg.Add(1)
		lock := <-p
		go func(build *build, lock bool) {
			defer wg.Done()
			handler(build)
			p <- lock
			callback <- build
		}(b, lock)
	}
}

// build an image
func builder(b *build) {
	log.Warnf("%s Building       %s", b.ID, b.prettyName())
	err := b.build()
	outStr := indent(string(b.Output), "  ")
	if err != nil {
		log.Errorf("%s Build failed   %s, b.ID, %s\n  >> Arguments: %s\n%s\n", b.ID, b.prettyName(), err, b.args(), outStr)
	}
}

// upload an image
func uploader(b *build) {
	log.Warnf("%s Uploading      %s", b.ID, b.prettyName())
	err := b.upload()
	outStr := indent(string(b.Output), "  ")
	if err != nil {
		log.Errorf("%s Upload failed  %s\n%s\n", b.ID, b.prettyName(), outStr)
	}
}

// prettyName
func (b *build) prettyName() string {
	return fmt.Sprintf("%s/%s/%s:%s", c.Registry, b.Namespace, b.Name, b.Tag)
}

// gather tags
func (b *build) tags() []string {
	tags := []string{fmt.Sprintf("%s/%s/%s:%s", c.Registry, b.Namespace, b.Name, b.Tag)}
	for _, name := range b.AdditionalNames {
		tags = append(tags, fmt.Sprintf("%s:%s", name, b.Tag))
	}
	return tags
}

// build command argument
func (b *build) args() []string {
	args := []string{"build", b.Name}
	for _, k := range b.KeyOrder {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", k, b.Scenario[k]))
	}
	for _, tag := range b.tags() {
		args = append(args, "-t", tag)
	}
	return args
}

// build image
func (b *build) build() (err error) {
	cmd := exec.Command(c.Command, b.args()...)
	_ = cmd.Wait()
	b.Output, err = cmd.CombinedOutput()
	return err
}

// upload image
func (b *build) upload() (err error) {
	for _, tag := range b.tags() {
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

// helper to indent strings
func indent(text string, prefix string) (out string) {
	for _, l := range strings.Split(text, "\n") {
		out += prefix + l + "\n"
	}
	return out
}

