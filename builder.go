package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	log "github.com/sirupsen/logrus"
)

type (
	BuildHandler func(*DockerBuild)

	// Builder starts up a worker for each step and builds the images
	Builder struct {
		parse  *Parser
		build  *Worker
		upload *Worker
		finish *Finisher
	}
)

func NewBuilder(builder, uploader, finisher BuildHandler) *Builder {
	inputc := make(chan *DockerBuild, 128)
	uploadc := make(chan *DockerBuild, 128)
	finishc := make(chan *DockerBuild, 128)

	parse := &Parser{
		wg:     &sync.WaitGroup{},
		output: inputc,
	}
	build := &Worker{
		name:    "build",
		wg:      &sync.WaitGroup{},
		input:   inputc,
		output:  uploadc,
		handler: builder,
	}
	upload := &Worker{
		name:    "upload",
		wg:      &sync.WaitGroup{},
		input:   uploadc,
		output:  finishc,
		handler: uploader,
	}
	finish := &Finisher{
		wg:      &sync.WaitGroup{},
		input:   finishc,
		handler: finisher,
	}

	return &Builder{
		parse:  parse,
		build:  build,
		upload: upload,
		finish: finish,
	}
}

func (b *Builder) Run(path string) error {
	// start builders in backgroud
	b.build.wg.Add(1)
	b.upload.wg.Add(1)
	b.finish.wg.Add(1)
	go b.upload.pool(128)
	go b.build.pool(128)
	go b.finish.Handle()

	// go to docker image folder
	oldPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("Failed to get current workdir %w", err)
	}
	err = os.Chdir(path)
	if err != nil {
		return fmt.Errorf("Failed to change directory to %s: %w", path, err)
	}

	changes := map[string]bool{}
	if !c.Dronetrigger && c.DiffOnly {
		changes, err = diff()
		if err != nil {
			return fmt.Errorf("unable to diff to generate diff: %s", err)
		}
	}
	noChanges := (len(changes) == 0)
	if noChanges {
		log.Warnf("No changes found")
	}

	// check for files
	err = filepath.Walk(".", func(file string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		dir := filepath.Dir(file)
		name := filepath.Base(dir)
		filename := filepath.Base(file)
		if filename != "Dockerfile" {
			return nil
		}

		// build if one of these match
		// * changed (per folder)
		// * run by dronetrigger (rebuilds all)
		// * no no changes found and diffonly is not set (rebuilds all)
		_, found := changes[dir]
		if c.Dronetrigger || (noChanges && !c.DiffOnly) {
			found = true
		}
		if found {
			err := b.parse.Parse(name)
			if err != nil {
				return fmt.Errorf("unable to parse file: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("unable to walk files: %w", err)
	}

	// wait for tasks to finish
	b.parse.WaitAndClose()
	b.build.WaitAndClose()
	b.upload.WaitAndClose()
	b.finish.Wait()

	// return to old working directory, required to run tests multiple times
	err = os.Chdir(oldPath)
	if err != nil {
		log.Fatalf("Failed to change directory to %s: %s", path, err)
	}

	return nil
}
