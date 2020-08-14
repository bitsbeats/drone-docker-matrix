package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

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
