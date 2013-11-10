package main

import (
	"log"
	"time"
	"strconv"
	"fmt"
	"errors"
	"github.com/dotcloud/docker"
	dcli "github.com/fsouza/go-dockerclient"
	"io"
	"io/ioutil"
	"os"
	"path"
)

type Runner struct {
	ContainerId string
	CodeDir     string
	Language    string
	Code        string
	OutStream   io.Writer
	ErrStream   io.Writer
	Uid			int
	UidPool     *UidPool
	client		*dcli.Client
}

const (
	STATUS_SUCCESS = 0
	STATUS_TIMED_OUT = -1
)

func NewRunner(client *dcli.Client, lang string, code string) *Runner {
	return &Runner{Language: lang, Code: code, client: client, Uid: 10000}
}

func (r *Runner) Run(timeout time.Duration) (int, error) {
	log.Println("Creating code directory")
	if err := r.createCodeDir(); err != nil {
		return 0, err
	}

	log.Println("Creating source file")
	srcFile, err := r.createSrcFile()
	if err != nil {
		return 0, err
	}

	if r.UidPool != nil {
		log.Println("Reserving uid")
		uid, err := r.UidPool.Reserve()
		if err != nil {
			return 0, err
		}
		log.Printf("Got uid %d\n", uid)
		r.Uid = uid
	}

	log.Println("Creating container")
	if err := r.createContainer(srcFile); err != nil {
		return 0, err
	}

	log.Println("Starting container")
	if err := r.startContainer(); err != nil {
		return 0, err
	}
	defer r.cleanup()

	log.Println("Streaming container logs")
	go func() {
		if err := r.streamLogs(); err != nil {
			log.Println(err)
		}
	}()

	log.Println("Waiting for container to finish")
	killed, status := r.waitForExit(timeout)
	if killed {
		log.Printf("Container exited with status %d\n", status)
		return STATUS_TIMED_OUT, nil
	}

	return STATUS_SUCCESS, nil
}

func (r *Runner) createCodeDir() error {
	dir, err := ioutil.TempDir("", "code-")
	r.CodeDir = dir
	return err
}

func (r *Runner) createSrcFile() (string, error) {
	ext, err := extForLanguage(r.Language)
	if err != nil {
		return "", err
	}

	fileName := fmt.Sprintf("prog.%s", ext)
	filePath := path.Join(r.CodeDir, fileName)
	f, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := f.WriteString(r.Code); err != nil {
		return "", err
	}

	return fileName, nil
}

func (r *Runner) createContainer(srcFile string) error {
	volPathOpts := docker.NewPathOpts()
	volPathOpts.Set("/code")
	uidStr := strconv.Itoa(r.Uid)
	config := &docker.Config{
		CpuShares:       1,
		Memory:			 50e6,
		Tty:             true,
		OpenStdin:       false,
		Volumes:         volPathOpts,
		Cmd:             []string{path.Join("/code", srcFile), uidStr},
		Image:           "runner",
		NetworkDisabled: true,
	}

	container, err := r.client.CreateContainer(config)
	if err != nil {
		return err
	}

	r.ContainerId = container.ID
	return nil
}

func (r *Runner) startContainer() error {
	if r.ContainerId == "" {
		return errors.New("Can't start a container before it is created")
	}

	hostConfig := &docker.HostConfig{
		Binds: []string{fmt.Sprintf("%s:/code", r.CodeDir)},
	}
	if err := r.client.StartContainer(r.ContainerId, hostConfig); err != nil {
		return err
	}

	return nil
}

func (r *Runner) waitForExit(timeoutMs time.Duration) (bool, int) {
	statusChan := make(chan int)
	go func() {
		if status, err := r.client.WaitContainer(r.ContainerId); err != nil {
			log.Println(err)
		} else {
			statusChan <- status
		}
	}()

	killed := false
	for {
		select {
		case status := <-statusChan:
			log.Println("Container exited by itself")
			return killed, status
		case <-time.After(time.Millisecond * timeoutMs):
			log.Println("Container timed out, killing")
			if err := r.client.StopContainer(r.ContainerId, 0); err != nil {
				log.Println(err)
			}
			killed = true
		}
	}
}

func (r *Runner) cleanup() {
	log.Println("Removing container")
	if err := r.client.RemoveContainer(r.ContainerId); err != nil {
		log.Printf("Couldn't remove container %s (%v)\n", r.ContainerId, err)
	}

	if r.UidPool != nil {
		log.Println("Releasing uid")
		if err := r.UidPool.Release(r.Uid); err != nil {
			log.Printf("Couldn't release uid %d (%v)\n", r.Uid, err)
		}
		r.Uid = 0
	}

	log.Println("Removing code dir")
	if err := os.RemoveAll(r.CodeDir); err != nil {
		log.Printf("Couldn't remove temp dir %s (%v)\n", r.CodeDir, err)
	}
}

func (r *Runner) streamLogs() error {
	if r.ContainerId == "" {
		return errors.New("Can't attach to a container before it is created")
	}

	attachOpts := dcli.AttachToContainerOptions{
		Container:    r.ContainerId,
		OutputStream: r.OutStream,
		ErrorStream:  r.ErrStream,
		Logs:         true,
		Stream:       true,
		Stdout:       true,
		Stderr:       true,
	}
	if err := r.client.AttachToContainer(attachOpts); err != nil {
		return err
	}

	return nil
}

func extForLanguage(lang string) (string, error) {
	switch lang {
	case "c":
		return "c", nil
	case "golang":
		return "go", nil
	case "perl":
		return "pl", nil
	case "python":
		return "py", nil
	case "ruby":
		return "rb", nil
	}
	return "", fmt.Errorf("Invalid language %v", lang)
}

