package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"ipfs-livestream/cliexec"
	"os"
	"os/exec"
	"strings"
)

type IPFSController struct {
	ipfsPath string
	ipgetPath string
	daemonProcess *os.Process

	cliexec.Controller
}

type PersonalId struct {
	ID              string   `json:"ID"`
	PublicKey       string   `json:"PublicKey"`
	Addresses       []string `json:"Addresses"`
	AgentVersion    string   `json:"AgentVersion"`
	ProtocolVersion string   `json:"ProtocolVersion"`
}

func NewIPFSController(ipfsPath, ipgetPath string) *IPFSController {
	return &IPFSController{ipfsPath, ipgetPath, nil, cliexec.Controller{}}
}

func genericError(err error, data []byte) error {
	return errors.New(err.Error() + " " + string(data))
}

func (c *IPFSController) GetResource(newName, address string, ipns bool) (err error) {
	if ipns {
		_, err = c.ExecutePath(c.ipgetPath, []string{"-o", newName, "/ipns/" + address})
	} else {
		_, err = c.ExecutePath(c.ipgetPath, []string{"-o", newName, address})
	}
	return err
}

func (c *IPFSController) PublishName(name string) error {
	data, err := c.ExecutePath(c.ipfsPath, []string{"name", "publish", "/ipfs/" + name, "--local"})
	if err != nil {
		return err
	}
	output := string(data)
	if strings.Index(output, "Published to") < 0 {
		return errors.New("publishing error: " + output)
	}
	return nil
}

func (c *IPFSController) PushFolder(path string) error {
	const mark = "added"
	const markLen = len(mark)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return err
	}

	data, err := c.ExecutePath(c.ipfsPath, []string{"add", "-r", path})
	if err != nil {
		return err
	}

	output := string(data)
	p := strings.LastIndex(output, "added")
	if p < 0 {
		return errors.New("processing error: " + output)
	}

	output = output[p + markLen + 1:]
	folderHashId := output[:strings.Index(output, " ")]
	return c.PublishName(folderHashId)
}

func (c *IPFSController) PushFile(path string) (string, error) {
	const mark = "added"
	const markLen = len(mark)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", err
	}

	data, err := c.ExecutePath(c.ipfsPath, []string{"add", path})
	if err != nil {
		return "", err
	}

	output := string(data)
	p := strings.LastIndex(output, "added")
	if p < 0 {
		return "", errors.New("processing error: " + output)
	}

	output = output[p + markLen + 1:]
	fileHashId := output[:strings.Index(output, " ")]
	return fileHashId, nil
}

func (c *IPFSController) GetId() (*PersonalId, error) {
	data, err := c.ExecutePath(c.ipfsPath, []string{"id"})
	if err != nil {
		return nil, err

	}
	id := &PersonalId{}
	return id, json.Unmarshal(data, id)
}

func (c *IPFSController) SaveBootstrapList(filename string) error {
	data, err := c.ExecutePath(c.ipfsPath, []string{"bootstrap", "list"})
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, data, os.ModePerm)
}

func (c *IPFSController) LoadBootstrapList(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return c.SetBootstrapList(strings.Split(string(data), "\n"))
}

func (c *IPFSController) ClearBootstrapList() error {
	args := []string{"bootstrap", "rm", "--all"}
	output, err := c.ExecutePath(c.ipfsPath, args)
	if err != nil {
		return genericError(err, output)
	}
	return nil
}

func (c *IPFSController) SetBootstrapList(list []string) error {
	err := c.ClearBootstrapList()
	if err != nil {
		return err
	}
	// add every bootstrap node to the ipfs
	args := []string{"bootstrap", "add", ""}
	for _, node := range list {
		if len(node) < 70 {
			continue
		}
		args[2] = node
		output, err := c.ExecutePath(c.ipfsPath, args)
		if err != nil {
			return genericError(err, output)
		}
	}
	return nil
}

func (c *IPFSController) StartDaemon() error {
	cmd := exec.Command(c.ipfsPath, []string{"daemon"}...)
	err := cmd.Start()
	if err != nil {
		return err
	}
	c.daemonProcess = cmd.Process
	return nil
}

func (c *IPFSController) StopDaemon() error {
	if c.daemonProcess != nil {
		err := c.daemonProcess.Kill()
		if err != nil {
			return err
		}
		c.daemonProcess = nil
	}
	return nil
}