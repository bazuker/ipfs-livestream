package cliexec

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"
)

type ControllerInterface interface {
	ExecutePath(path string, args []string) ([]byte, error)
	ExecutePathWithDuration(path string, args []string, maxDuration time.Duration) ([]byte, error)
}

type Controller struct {
}

func (c *Controller) ExecutePath(path string, args []string) ([]byte, error) {
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(path, args...)
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return nil, errors.New(fmt.Sprint(err) + ": " + stderr.String())
	}
	return out.Bytes(), nil
}

func (c *Controller) ExecutePathWithDuration(path string, args []string, maxDuration time.Duration) ([]byte, error) {
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(path, args...)
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Start()
	if err != nil {
		return nil, errors.New(fmt.Sprint(err) + ": " + stderr.String())
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case <-time.After(maxDuration):
		if err := cmd.Process.Signal(os.Interrupt); err != nil {
			log.Println("failed to interrupt process:", err.Error())
		}
	case err := <-done:
		if err != nil {
			log.Println("process finished with error", err.Error())
		}
	}
	return out.Bytes(), nil
}