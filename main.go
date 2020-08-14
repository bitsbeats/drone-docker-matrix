package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/drone/envsubst"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type (
	// Configuration
	config struct {
		// Registry is the registry to upload the images to
		Registry string `envconfig:"REGISTRY"`
		// PushGateway is the URL to Prometheus Pushgateway for metrics
		PushGateway string `envconfig:"PUSHGATEWAY`

		BuildPoolSize  int `envconfig:"BUILD_POOL_SIZE" default:"4"`
		UploadPoolSize int `envconfig:"UPLOAD_POOL_SIZE" default:"4"`

		// DefaultNamespace is the Namespace to use if not specified in
		// `docker-matrix.yml` (default: `images`)
		DefaultNamespace string `envconfig:"DEFAULT_NAMESPACE" default:"images"`
		// TagName is the default tag name
		TagName string `envconfig:"TAG_NAME" default:"latest"`
		// TagBuildID generates an additional tag `tagname-b<ID>` for
		// each tag, skipped if empty
		TagBuildID string `envconfig:"TAG_BUILD_ID"`
		// SkipUpload skips the upload to registry, useful for testing
		SkipUpload bool `envconfig:"SKIP_UPLOAD" default:"false"`
		// Pull trues to pull all docker images
		Pull bool `envconfig:"PULL" default:"true"`

		// Workdir changes the working directory before calculating the
		// matrix
		Workdir string `envconfig:"WORKDIR" default:"."`
		// DiffOnly builds only changes, if no change is detected
		// nothing will be build
		DiffOnly bool `envconfig:"DIFF_ONLY" default:"true"`
		// Dronetigger builds all images, regardless of other options
		Dronetrigger bool `envconfig:"DRONETRIGGER" default:"false"`
		// Command is the command to build dockerimages with
		Command string `default:"docker"`
		// Debug enables debuglogging
		Debug bool `envconfig:"DEBUG" default:"false"`

		// Time is set during startup and is used as Label on the
		// indiviual images
		Time time.Time
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

		// AsLatest contains the that that will additionaly be tagged as latest
		AsLatest string `yaml:"as_latest"`

		// CustomPath allowes to overwrite the path of the docker context
		CustomPath string `yaml:"custom_path"`

		// Dockerfile alles to specify a custom Dockerfile
		CustomDockerfile string `yaml:"custom_dockerfile" default:"Dockerfile"`
	}
)

var c config

func main() {
	// configuration
	log.SetFormatter(&log.TextFormatter{ForceColors: true})
	err := envconfig.Process("plugin", &c)
	if err != nil {
		log.Fatal(err.Error())
	}
	if c.BuildPoolSize < 1 || c.UploadPoolSize < 1 {
		log.Fatalf("PoolSize may not be smaller than 1: BuildPoolSize: %d, UploadPoolSize: %d", c.BuildPoolSize, c.UploadPoolSize)
	}
	if c.Registry == "" {
		log.Fatal("Please specify a registry.")
	}
	c.TagBuildID, err = envsubst.EvalEnv(c.TagBuildID)
	if err != nil {
		log.Fatal(err.Error())
	}
	if c.Debug {
		log.SetLevel(log.DebugLevel)
	}
	c.Time = time.Now()
	log.Infof("Configuration: %+v", c)

	// run
	b := NewBuilder(
		builder,
		uploader,
		func(b *Build) {
			log.Infof("Done           %s", b.prettyName())
		},
	)
	b.Run(c.Workdir)
}

// build an image
func builder(b *Build) {
	err := b.build()
	outStr := indent(string(b.Output), "  ")
	if err != nil {
		log.Errorf("Build failed   %s, %s\n  >> Arguments: %s\n%s\n", b.prettyName(), err, b.args(), outStr)
	} else {
		log.Debugf("Build success  %s\n  >> Arguments: %s\n%s\n", b.prettyName(), b.args(), outStr)
	}
}

// upload an image
func uploader(b *Build) {
	if c.SkipUpload {
		return
	}
	err := b.upload()
	outStr := indent(string(b.Output), "  ")
	if err != nil {
		log.Errorf("Upload failed  %s\n%s\n", b.prettyName(), outStr)
	} else {
		log.Debugf("Upload success %s\n%s\n", b.prettyName(), outStr)
	}
	return
}

// helper to indent strings
func indent(text string, prefix string) (out string) {
	for _, l := range strings.Split(text, "\n") {
		out += prefix + l + "\n"
	}
	return out
}

// git diff
func diff() (dirs map[string]bool, err error) {
	before := os.Getenv("DRONE_COMMIT_BEFORE")
	ref := os.Getenv("DRONE_COMMIT_REF")
	dirs = map[string]bool{}

	if strings.HasPrefix(ref, "refs/pull/") {
		// pull request
		before = "origin/master"
	} else if before != "" {
		// normal commit, usually ref is a sha
		before = strings.TrimPrefix(before, "refs/")
	} else {
		// empty history, skipping build
		// TODO: remove this
		return nil, nil
		//return nil, fmt.Errorf("unable to fetch previos commit from DRONE_COMMIT_REF")
	}

	// changes since last commit
	cmd := exec.Command("git", "diff", "--name-only", before)
	_ = cmd.Wait()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	// working directory changes
	_, inDrone := os.LookupEnv("DRONE")
	if !inDrone && len(out) == 0 {
		log.Warn("No changes found, looking for uncommited changes.")
		cmd = exec.Command("git", "status", "-u", "--porcelain")
		_ = cmd.Wait()
		out2, err := cmd.CombinedOutput()
		if err != nil {
			return nil, err
		}
		for _, line := range bytes.Split(out2, []byte("\n")) {
			if len(line) > 3 {
				line = line[3:]
				out = append(out, line...)
				out = append(out, []byte("\n")...)
			}
		}
	}

	for _, file := range strings.Split(string(out), "\n") {
		split := strings.Split(file, string(os.PathSeparator))
		if len(split) > 0 {
			name := split[0]
			if _, err := os.Stat(filepath.Join(name, "Dockerfile")); err == nil {
				dirs[name] = true
			}
		}
	}

	log.Infof("Diff mode enabled (%s), building following images: %v", before, dirs)
	return dirs, nil
}
