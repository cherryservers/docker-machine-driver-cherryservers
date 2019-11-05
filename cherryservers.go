package main

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/pkg/errors"

	"github.com/cherryservers/cherrygo"
	"github.com/docker/machine/libmachine/drivers"
	"github.com/docker/machine/libmachine/log"
	"github.com/docker/machine/libmachine/mcnflag"
	"github.com/docker/machine/libmachine/mcnutils"
	"github.com/docker/machine/libmachine/ssh"
	"github.com/docker/machine/libmachine/state"
)

const (
	defaultSSHPort = 22
	defaultSSHUser = "root"
	defaultImage   = "Ubuntu 16.04 64bit"
	defaultRegion  = "EU-East-1"
	defaultPlan    = "94"
)

// Driver struct
type Driver struct {
	*drivers.BaseDriver
	AuthToken           string
	ProjectID           string
	Hostname            string
	ServerID            string
	DropletName         string
	Image               string
	Region              string
	SSHKeyID            string
	SSHKeyLabel         string
	SSHKey              string
	ExistingSSHKeyPath  string
	ExistingSSHKeyLabel string
	Plan                string
	PrivateNetworking   bool
	UserDataFile        string
	Tags                string
}

// GetCreateFlags registers the flags this driver adds to
// "docker hosts create"
func (d *Driver) GetCreateFlags() []mcnflag.Flag {
	return []mcnflag.Flag{
		mcnflag.StringFlag{
			EnvVar: "CHERRYSERVERS_AUTH_TOKEN",
			Name:   "cherryservers-auth-token",
			Usage:  "Cherry Servers auth token",
		},
		mcnflag.StringFlag{
			EnvVar: "CHERRYSERVERS_PROJECT_ID",
			Name:   "cherryservers-project-id",
			Usage:  "Cherry Servers project id",
		},
		mcnflag.StringFlag{
			EnvVar: "CHERRYSERVERS_HOSTNAME",
			Name:   "cherryservers-hostname",
			Usage:  "Cherry Servers server hostname",
		},
		mcnflag.StringFlag{
			EnvVar: "CHERRYSERVERS_SSH_USER",
			Name:   "cherryservers-ssh-user",
			Usage:  "SSH username",
			Value:  defaultSSHUser,
		},
		mcnflag.StringFlag{
			EnvVar: "CHERRYSERVERS_EXISTING_SSH_KEY_LABEL",
			Name:   "cherryservers-existing-ssh-key-label",
			Usage:  "SSH key label",
			Value:  "none",
		},
		mcnflag.StringFlag{
			EnvVar: "CHERRYSERVERS_EXISTING_SSH_KEY_PATH",
			Name:   "cherryservers-existing-ssh-key-path",
			Usage:  "SSH private key path ",
		},
		mcnflag.IntFlag{
			EnvVar: "CHERRYSERVERS_SSH_PORT",
			Name:   "cherryservers-ssh-port",
			Usage:  "SSH port",
			Value:  defaultSSHPort,
		},
		mcnflag.StringFlag{
			EnvVar: "CHERRYSERVERS_IMAGE",
			Name:   "cherryservers-image",
			Usage:  "Cherry Servers Image",
			Value:  defaultImage,
		},
		mcnflag.StringFlag{
			EnvVar: "CHERRYSERVERS_REGION",
			Name:   "cherryservers-region",
			Usage:  "Cherry Servers region",
			Value:  defaultRegion,
		},
		mcnflag.StringFlag{
			EnvVar: "CHERRYSERVERS_PLAN",
			Name:   "cherryservers-plan",
			Usage:  "Cherry Servers plan",
			Value:  defaultPlan,
		},
		mcnflag.StringFlag{
			EnvVar: "CHERRYSERVERS_USERDATA",
			Name:   "cherryservers-userdata",
			Usage:  "path to file with cloud-init user-data",
		},
		mcnflag.StringFlag{
			EnvVar: "CHERRYSERVERS_TAGS",
			Name:   "cherryservers-tags",
			Usage:  "comma-separated list of tags to apply to the server",
		},
	}
}

// NewDriver returns a new driver
func NewDriver(hostName, storePath string) *Driver {
	return &Driver{
		BaseDriver: &drivers.BaseDriver{},
	}
}

// GetSSHHostname gets IP address
func (d *Driver) GetSSHHostname() (string, error) {

	return d.GetIP()
}

// DriverName returns the name of the driver
func (d *Driver) DriverName() string {
	return "cherryservers"
}

// GetState returns the state of the server
func (d *Driver) GetState() (state.State, error) {

	client := d.getClient()
	server, _, err := client.Server.List(d.ServerID)
	if err != nil {
		return state.Error, err
	}
	switch server.State {
	case "provisioning":
		return state.Starting, nil
	case "active":
		return state.Running, nil
	case "terminating":
		return state.Stopped, nil
	}
	return state.None, nil
}

// SetConfigFromFlags sets flags
func (d *Driver) SetConfigFromFlags(flags drivers.DriverOptions) error {
	d.AuthToken = flags.String("cherryservers-auth-token")
	d.ProjectID = flags.String("cherryservers-project-id")
	d.Hostname = flags.String("cherryservers-hostname")
	d.Image = flags.String("cherryservers-image")
	d.Plan = flags.String("cherryservers-plan")
	d.Region = flags.String("cherryservers-region")
	d.SSHUser = flags.String("cherryservers-ssh-user")
	d.SSHPort = flags.Int("cherryservers-ssh-port")
	d.ExistingSSHKeyLabel = flags.String("cherryservers-existing-ssh-key-label")
	d.ExistingSSHKeyPath = flags.String("cherryservers-existing-ssh-key-path")
	d.Tags = flags.String("cherryservers-tags")

	d.SetSwarmConfigFromFlags(flags)

	if d.AuthToken == "" {
		return fmt.Errorf("cherryservers driver requires the --cherryservers-auth-token option")
	}

	return nil
}

func (d *Driver) getKey(label string) (keyID int, keyFP string, err error) {

	sshkeys, _, err := d.getClient().SSHKeys.List()

	for _, s := range sshkeys {
		if label == s.Label {
			keyID = s.ID
			keyFP = s.Fingerprint
		}
	}

	return keyID, keyFP, err
}

func getFingerPrint(pubKey []byte) (fp string, err error) {

	parts := strings.Fields(string(pubKey))
	if len(parts) < 2 {
		return "", errors.New("bad key")
	}

	b, _ := base64.StdEncoding.DecodeString(string(parts[1]))
	h := md5.New()

	io.WriteString(h, string(b))

	return fmt.Sprintf("%x", h.Sum(nil)), err
}

// PreCreateCheck check various args
func (d *Driver) PreCreateCheck() error {

	if d.ExistingSSHKeyLabel != "none" {
		if d.ExistingSSHKeyPath == "" {
			return errors.New("by specifying an existing ssh key label you must specify path to key as well")
		}

		key, remoteKeyFP, err := d.getKey(d.ExistingSSHKeyLabel)
		if err != nil {
			return errors.Wrap(err, "could not find such key")
		}

		// Remove semicolons from Fingerprint
		remoteKeyFP = strings.Replace(remoteKeyFP, ":", "", -1)

		// Check if we can read existing public key
		buf, err := ioutil.ReadFile(d.ExistingSSHKeyPath + ".pub")
		if err != nil {
			return errors.Wrap(err, "could not read ssh public key")
		}

		exKeyFP, err := getFingerPrint(buf)

		if exKeyFP != remoteKeyFP {
			return errors.Errorf("remote key %s does not match local key %s", remoteKeyFP, exKeyFP)
		}

		d.SSHKeyID = strconv.Itoa(key)
	}

	return nil
}

// copySSHKeyPair copies specified key to machine folder
func (d *Driver) copySSHKeyPair(src string) error {
	if err := mcnutils.CopyFile(src, d.GetSSHKeyPath()); err != nil {
		return errors.Wrap(err, "could not copy ssh key")
	}

	if err := mcnutils.CopyFile(src+".pub", d.GetSSHKeyPath()+".pub"); err != nil {
		return errors.Wrap(err, "could not copy ssh public key")
	}

	if err := os.Chmod(d.GetSSHKeyPath(), 0600); err != nil {
		return errors.Wrap(err, "could not set permissions on the ssh key")
	}

	return nil
}

// Create will create a new server with Cherry Servers
func (d *Driver) Create() error {

	log.Infof("Deploying Cherry Servers node...")

	client := d.getClient()

	var sLabels []string

	if d.ExistingSSHKeyPath != "" {
		log.Debugf("Try to copy existing SSH key to machine's store path")
		if err := d.copySSHKeyPair(d.ExistingSSHKeyPath); err != nil {
			return errors.Wrap(err, "could not nopy ssh key pair")
		}
	} else {
		log.Debugf("Generating new SSH key...")

		log.Infof("Creating SSH key...")
		if error := ssh.GenerateSSHKey(d.GetSSHKeyPath()); error != nil {
			return error
		}
	}

	log.Debugf("Try to create ssh key in API")

	// In case of existing key
	if d.ExistingSSHKeyLabel == "none" {

		// Check if we can read existing public key
		buf, err := ioutil.ReadFile(d.GetSSHKeyPath() + ".pub")
		if err != nil {
			return errors.Wrap(err, "could not read ssh public key")
		}

		client := d.getClient()

		// Use machine name as label for API's key
		var label string
		label = d.MachineName

		sshCreateRequest := cherrygo.CreateSSHKey{
			Label: label,
			Key:   string(buf),
		}

		sshkey, _, err := client.SSHKey.Create(&sshCreateRequest)
		if err != nil {
			return err
		}

		keyIDString := strconv.Itoa(sshkey.ID)

		d.SSHKeyID = keyIDString

		sLabels = append(sLabels, d.SSHKeyID)

	} else {

		sLabels = append(sLabels, d.SSHKeyID)
	}

	var ipaddresses = make([]string, 0)

	addServerRequest := cherrygo.CreateServer{
		ProjectID:   d.ProjectID,
		Hostname:    d.Hostname,
		Image:       d.Image,
		Region:      d.Region,
		SSHKeys:     sLabels,
		IPAddresses: ipaddresses,
		PlanID:      d.Plan,
		//UserData:    userData,
		//Tags:        tags,
	}

	server, _, err := client.Server.Create(d.ProjectID, &addServerRequest)
	if err != nil {
		return err
	}

	serverUID := strconv.Itoa(server.ID)

	d.ServerID = serverUID

	log.Info("Wait for server to be deployed")
	err = waitForServer(d)

	return nil
}

// GetURL gets url
func (d *Driver) GetURL() (string, error) {

	if err := drivers.MustBeRunning(d); err != nil {
		return "", err
	}

	ip, err := d.GetIP()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("tcp://%s", net.JoinHostPort(ip, "2376")), nil
}

// Start powers server on
func (d *Driver) Start() error {
	_, _, err := d.getClient().Server.PowerOn(d.ServerID)
	return err
}

// Stop powers server off
func (d *Driver) Stop() error {
	_, _, err := d.getClient().Server.PowerOff(d.ServerID)
	return err
}

// Restart reboots the server
func (d *Driver) Restart() error {
	_, _, err := d.getClient().Server.Reboot(d.ServerID)
	return err
}

// Kill power offs the server
func (d *Driver) Kill() error {
	_, _, err := d.getClient().Server.PowerOff(d.ServerID)
	return err
}

// Remove removes the servers
func (d *Driver) Remove() error {

	serverDeleteRequest := cherrygo.DeleteServer{ID: d.ServerID}

	_, _, err := d.getClient().Server.Delete(&serverDeleteRequest)
	if err != nil {
		return err
	}

	return nil
}

func (d *Driver) getClient() *cherrygo.Client {
	client := cleanhttp.DefaultClient()
	cherryClient := cherrygo.NewClientWithAuthVar(client, d.AuthToken)

	return cherryClient
}

func waitForServer(d *Driver) error {

	client := d.getClient()

	for i := 1; i < 300; i++ {

		time.Sleep(time.Second * 10)

		server, _, err := client.Server.List(d.ServerID)

		if err != nil {
			err = fmt.Errorf("timed out waiting for active device: %v", d.ServerID)
		}

		for _, ip := range server.IPAddresses {
			if ip.Type == "primary-ip" {
				if ip.Address != "" {
					if server.State == "active" {
						d.IPAddress = ip.Address
						return nil
					}

				}
			}
		}
	}

	err := fmt.Errorf("timed out waiting for active device: %v", d.ServerID)

	return err
}
