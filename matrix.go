package main

import (
	"fmt"

	"github.com/drone/envsubst"
	"github.com/segmentio/ksuid"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type (
	// Matrix that describes all arguments required to build the dockerfile.
	Matrix struct {
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
		Append []yaml.MapSlice `yaml:"append"`

		// Namespace: namespace for the image
		Namespace string `yaml:"namespace"`

		// AdditionalNames is a comma seperated list of tag names to upload as
		//
		//   additional_names: bitsbeats/drone-docker-matrix
		AdditionalNames []string `yaml:"additional_names"`

		// AsLatest contains the that that will additionaly be tagged as latest
		AsLatest string `yaml:"as_latest"`

		// CustomPath allowes to overwrite the path of the docker context
		CustomPath string `yaml:"custom_path"`

		// CustomDockerfile allowes to specify a custom Dockerfile
		CustomDockerfile string `yaml:"custom_dockerfile" default:"Dockerfile"`
	}
)

type (
	MatrixMultiplyItem struct {
		Name string
		Values []string
	}
)

func (m Matrix) getEnvSubstedMultiply(buildID ksuid.KSUID) []MatrixMultiplyItem {
	result := []MatrixMultiplyItem{}
	for _, item := range m.Multiply {
		argument := fmt.Sprintf("%v", item.Key)
		values := []string{}
		for _, value := range item.Value.([]interface{}) {
			stringValue := fmt.Sprintf("%v", value)
			parsedValue, err := envsubst.EvalEnv(stringValue)
			if err != nil {
				log.Errorf("%s unable to envsubst %s -> %s: %s", buildID, argument, stringValue, err)
				continue
			}
			values = append(values, parsedValue)
		}
		result = append(result, MatrixMultiplyItem{
			Name: argument,
			Values: values,
		})
	}
	return result
}
