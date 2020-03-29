package main

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/whilp/git-urls"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

// Environment contains information about git repository
type Environment struct {
	root        string
	branch      string
	author      string
	project     string
	initBranch  sync.Once
	initAuthor  sync.Once
	initProject sync.Once
}

// NewEnvironment creates new instance of Environment struct
func NewEnvironment(root string) *Environment {
	absolutePath, err := filepath.Abs(root)
	if err != nil {
		log.Printf("Error when setting env root: %v", err)
		absolutePath = root
	}
	env := &Environment{
		root: absolutePath,
	}
	go func() {
		log.Printf("Current root is %v", env.root)
		log.Printf("Current branch is %v", env.Branch())
		log.Printf("Current author is %v", env.Author())
		log.Printf("Current project is %v", env.Project())
	}()
	return env
}

func sliceWithoutGitDir(slice []string) []string {
	newEnv := make([]string, 0, len(slice))
	for _, s := range slice {
		if strings.HasPrefix(strings.ToUpper(s), "GIT_DIR") {
			continue
		}
		newEnv = append(newEnv, s)
	}
	return newEnv
}

// Run executes a command in the environment's root
func (env *Environment) Run(cmd string, arg ...string) string {
	command := exec.Command(cmd, arg...)
	// setting working directory here breaks GIT_DIR variable
	command.Dir = env.root
	// so we need to remove this variable from environment
	command.Env = sliceWithoutGitDir(os.Environ())

	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()
	if err != nil {
		log.Printf("Command run error: %s", err)
		log.Printf("Command stderr: %s", string(stderr.Bytes()))
		return ""
	}

	outStr := string(stdout.Bytes())
	return strings.TrimSpace(outStr)
}

// Branch returns current git branch
func (env *Environment) Branch() string {
	env.initBranch.Do(func() {
		env.branch = env.Run("git", "rev-parse", "--abbrev-ref", "HEAD")
	})
	return env.branch
}

// Author returns current git author
func (env *Environment) Author() string {
	env.initAuthor.Do(func() {
		env.author = env.Run("git", "config", "user.name")
	})
	return env.author
}

// Project returns current git project name
func (env *Environment) Project() string {
	env.initProject.Do(func() {
		project := env.Run("git", "rev-parse", "--show-toplevel")
		env.project = filepath.Base(project)
	})
	return env.project
}

// RefBranchName returns the branch name of a reference.
// It assumes that the ref has a branch type.
func refBranchName(ref *plumbing.Reference) string {
	return refBranchNameStr(ref.String())
}

// RefBranchNameStr returns the branch name of a reference string.
// It assumes that the ref has a branch type.
func refBranchNameStr(str string) string {
	parts := strings.Split(str, "/")
	return strings.Join(parts[2:], "/")
}

func getRemoteURLPath(path string) (string, error) {
	if path == "" {
		path = "."
	}
	// We instantiate a new repository targeting the given path (the .git folder)
	r, err := git.PlainOpen(path)
	if err != nil {
		return "", err
	}
	cfg, err := r.Config()
	if err != nil {
		return "", err
	}
	g, err := giturls.Parse(cfg.Remotes["origin"].URLs[0])
	if err != nil {
		return "", err
	}
	return strings.Replace(g.Path, ".git", "", -1), nil
}
