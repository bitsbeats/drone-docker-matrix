package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"time"

	"net/http"

	"github.com/drone/envsubst"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

type (
	// Configuration
	config struct {
		// Registry is the registry to upload the images to
		Registry string `envconfig:"REGISTRY"`
		// PushGateway is the URL to Prometheus Pushgateway for metrics
		PushGateway string `envconfig:"PUSHGATEWAY" default:""`

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

)

var c config

func main() {
	// configuration
	log.SetFormatter(&log.TextFormatter{ForceColors: true})
	err := envconfig.Process("plugin", &c)
	if err != nil {
		log.Fatalf("unable to parse environment: %s", err)
	}
	if c.BuildPoolSize < 1 || c.UploadPoolSize < 1 {
		log.Fatalf("PoolSize may not be smaller than 1: BuildPoolSize: %d, UploadPoolSize: %d", c.BuildPoolSize, c.UploadPoolSize)
	}
	if c.Registry == "" {
		log.Fatalf("Please specify a registry.")
	}
	c.TagBuildID, err = envsubst.EvalEnv(c.TagBuildID)
	if err != nil {
		log.Fatal(err)
	}
	if c.Debug {
		log.SetLevel(log.DebugLevel)
	}
	c.Time = time.Now()
	log.Infof("Configuration: %+v", c)

	// log info
	sysinfo := exec.Command(c.Command, "system", "info")
	sysinfo.Stdout = os.Stdout
	err = sysinfo.Run()
	if err != nil {
		log.Fatal(err)
	}

	// run
	b := NewBuilder(
		builder,
		uploader,
		finisher,
	)
	err = b.Run(c.Workdir)
	if err != nil {
		log.Fatal(err)
	}
}

// build an image
func builder(b *DockerBuild) {
	err := b.build()
	outStr := indent(string(b.Output), "  ")
	if err != nil {
		b.Error = err
		log.Errorf("Build failed   %s, %s\n  >> Arguments: %s\n%s\n", b.prettyName(), err, b.args(), outStr)
		return
	}
	log.Debugf("Build success  %s\n  >> Arguments: %s\n%s\n", b.prettyName(), b.args(), outStr)
}

// upload an image
func uploader(b *DockerBuild) {
	// skip all uploads even if only a single build failes
	if b.Error != nil {
		return
	}

	if c.SkipUpload {
		return
	}
	err := b.upload()
	outStr := indent(string(b.Output), "  ")
	if err != nil {
		b.Error = err
		log.Errorf("Upload failed  %s\n%s\n", b.prettyName(), outStr)
		return
	}
	log.Debugf("Upload success %s\n%s\n", b.prettyName(), outStr)
}

// finisher is called after an image is uploaded
func finisher(b *DockerBuild) {
	log.Infof("Done           %s", b.prettyName())

	// notify pushgateway if set
	if c.PushGateway != "" {
		// skip update if any of the tags failed
		if b.Error != nil {
			return
		}

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
}
