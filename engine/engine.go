// Copyright 2020 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package engine

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"time"

	"github.com/drone-runners/drone-runner-macstadium/internal/orka"
	"github.com/drone/runner-go/logger"
	"github.com/drone/runner-go/pipeline/runtime"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const networkTimeout = time.Minute * 10

// Engine implements a pipeline engine.
type Engine struct {
	client   *orka.Client
	username string
	password string
}

// New returns a new engine.
func New(client *orka.Client) (*Engine, error) {
	return &Engine{client: client}, nil
}

// Setup the pipeline environment.
func (e *Engine) Setup(ctx context.Context, specv runtime.Spec) error {
	spec := specv.(*Spec)

	logger.FromContext(ctx).
		WithField("id", spec.Name).
		Debug("create the vm config")

	// create the vm configuration.
	_, err := e.client.Create(ctx, &orka.Config{
		Name:  spec.Name,
		Image: spec.Settings.Image,
		CPU:   spec.Settings.Compute,
		VCPU:  spec.Settings.Compute,
	})
	if err != nil {
		logger.FromContext(ctx).
			WithError(err).
			WithField("id", spec.Name).
			Debug("failed to create the vm config")
		return err
	}

	logger.FromContext(ctx).
		WithField("id", spec.Name).
		Debug("provision the vm")

	// provision the virtual machine and return an
	// active ssh client connection.
	client, err := e.createRetry(ctx, spec)
	if client != nil {
		defer client.Close()
	}
	if err != nil {
		logger.FromContext(ctx).
			WithError(err).
			WithField("id", spec.Name).
			Debug("failed to provision the vm")
		return err
	}

	clientftp, err := sftp.NewClient(client)
	if err != nil {
		logger.FromContext(ctx).
			WithError(err).
			WithField("ip", spec.ip).
			WithField("id", spec.Name).
			Debug("failed to create sftp client")
		return err
	}
	defer clientftp.Close()

	// the pipeline specification may define global folders, such
	// as the pipeline working directory, wich must be created
	// before pipeline execution begins.
	for _, file := range spec.Files {
		if file.IsDir == false {
			continue
		}
		err = mkdir(clientftp, file.Path, file.Mode)
		if err != nil {
			logger.FromContext(ctx).
				WithError(err).
				WithField("path", file.Path).
				Error("cannot create directory")
			return err
		}
	}

	// the pipeline specification may define global files such
	// as authentication credentials that should be uploaded
	// before pipeline execution begins.
	for _, file := range spec.Files {
		if file.IsDir == true {
			continue
		}
		err = upload(clientftp, file.Path, file.Data, file.Mode)
		if err != nil {
			logger.FromContext(ctx).
				WithError(err).
				Error("cannot write file")
			return err
		}
	}

	logger.FromContext(ctx).
		WithField("ip", spec.ip).
		WithField("id", spec.Name).
		Debug("vm configuration complete")
	return nil
}

// Destroy the pipeline environment.
func (e *Engine) Destroy(ctx context.Context, specv runtime.Spec) error {
	spec := specv.(*Spec)
	if spec.ip == "" {
		return nil
	}
	logger.FromContext(ctx).
		WithField("ip", spec.ip).
		WithField("id", spec.Name).
		Debug("deleting vm")
	_, err := e.client.Delete(ctx, spec.Name)
	return err
}

// Run runs the pipeline step.
func (e *Engine) Run(ctx context.Context, specv runtime.Spec, stepv runtime.Step, output io.Writer) (*runtime.State, error) {
	spec := specv.(*Spec)
	step := stepv.(*Step)

	client, err := dial(
		spec.ip,
		spec.Settings.Username,
		spec.Settings.Password,
	)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	clientftp, err := sftp.NewClient(client)
	if err != nil {
		return nil, err
	}
	defer clientftp.Close()

	// unlike os/exec there is no good way to set environment
	// the working directory or configure environment variables.
	// we work around this by pre-pending these configurations
	// to the pipeline execution script.
	for _, file := range step.Files {
		w := new(bytes.Buffer)
		writeWorkdir(w, step.WorkingDir)
		writeSecrets(w, "posix", step.Secrets)
		writeEnviron(w, "posix", step.Envs)
		w.Write(file.Data)
		err = upload(clientftp, file.Path, w.Bytes(), file.Mode)
		if err != nil {
			logger.FromContext(ctx).
				WithError(err).
				WithField("path", file.Path).
				Error("cannot write file")
			return nil, err
		}
	}

	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	session.Stdout = output
	session.Stderr = output
	cmd := step.Command + " " + strings.Join(step.Args, " ")

	log := logger.FromContext(ctx)
	log.Debug("ssh session started")

	done := make(chan error)
	go func() {
		done <- session.Run(cmd)
	}()

	select {
	case err = <-done:
	case <-ctx.Done():
		// BUG(bradrydzewski): openssh does not support the signal
		// command and will not signal remote processes. This may
		// be resolved in openssh 7.9 or higher. Please subscribe
		// to https://github.com/golang/go/issues/16597.
		if err := session.Signal(ssh.SIGKILL); err != nil {
			log.WithError(err).Debug("kill remote process")
		}

		log.Debug("ssh session killed")
		return nil, ctx.Err()
	}

	state := &runtime.State{
		ExitCode:  0,
		Exited:    true,
		OOMKilled: false,
	}
	if err != nil {
		state.ExitCode = 255
	}
	if exiterr, ok := err.(*ssh.ExitError); ok {
		state.ExitCode = exiterr.ExitStatus()
	}

	log.WithField("ssh.exit", state.ExitCode).
		Debug("ssh session finished")
	return state, err
}

// Ping pings the underlying runtime to verify connectivity.
func (e *Engine) Ping(ctx context.Context) error {
	_, err := e.client.CheckToken(ctx)
	return err
}

//
// helper functions
//

func (e *Engine) createRetry(ctx context.Context, spec *Spec) (*ssh.Client, error) {
	client, err := e.create(ctx, spec)
	if err == nil {
		return client, nil
	}

	ctx, cancel := context.WithTimeout(ctx, time.Hour)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		client, err := e.create(ctx, spec)
		if err == nil {
			return client, nil
		}

		switch {
		case strings.Contains(err.Error(), "No available nodes"):
		case strings.Contains(err.Error(), "network is unreachable"):
		default:
			return nil, err
		}

		logger.FromContext(ctx).
			WithField("ip", spec.ip).
			WithField("id", spec.Name).
			Trace("retry to deploy the vm")

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Minute):
		}
	}
}

func (e *Engine) create(ctx context.Context, spec *Spec) (*ssh.Client, error) {
	logger.FromContext(ctx).
		WithField("id", spec.Name).
		Debug("deploy the vm")

	deploy, err := e.client.Deploy(ctx, spec.Name)
	if err != nil {
		logger.FromContext(ctx).
			WithError(err).
			WithField("id", spec.Name).
			Debug("failed to deploy the vm")
		return nil, err
	}

	// snapshot the ip address and port.
	spec.ip = deploy.IP + ":" + deploy.SSHPort

	logger.FromContext(ctx).
		WithField("id", spec.Name).
		WithField("ip", spec.ip).
		Debug("successfully deployed the vm")

	logger.FromContext(ctx).
		WithField("ip", spec.ip).
		WithField("id", spec.Name).
		Trace("dialing the vm")

	// establish an ssh connection with the server instance
	// to setup the build environment (upload build scripts, etc)
	client, err := dialRetry(ctx, spec)
	if err == nil {
		logger.FromContext(ctx).
			WithField("ip", spec.ip).
			WithField("id", spec.Name).
			Trace("successfully dialed the vm")
		return client, nil
	}

	logger.FromContext(ctx).
		WithField("ip", spec.ip).
		WithField("id", spec.Name).
		Trace("failed to dial the vm")

	// if the vm fails to properly deploy it is destroyed
	// and retried. if destroying the vm fails the
	// the error is ignored, since this should not prevent
	// subsequent retries.
	_ = e.Destroy(ctx, spec)

	return nil, err

	// ctx, cancel := context.WithTimeout(ctx, networkTimeout)
	// defer cancel()
	// for {
	// 	select {
	// 	case <-ctx.Done():
	// 		return nil, ctx.Err()
	// 	default:
	// 	}
	// 	logger.FromContext(ctx).
	// 		WithField("ip", spec.ip).
	// 		WithField("id", spec.Name).
	// 		Trace("dialing the vm")
	// 	client, err = dial(
	// 		spec.ip,
	// 		spec.Settings.Username,
	// 		spec.Settings.Password,
	// 	)
	// 	if err == nil {
	// 		return client, nil
	// 	}
	// 	logger.FromContext(ctx).
	// 		WithError(err).
	// 		WithField("ip", spec.ip).
	// 		WithField("id", spec.Name).
	// 		Trace("failed to dial vm")

	// 	if client != nil {
	// 		client.Close()
	// 	}

	// 	select {
	// 	case <-ctx.Done():
	// 		return nil, ctx.Err()
	// 	case <-time.After(time.Second * 10):
	// 	}
	// }
}

// helper function configures and dials the ssh server and
// retries until a connection is established or a timeout
// is reached.
func dialRetry(ctx context.Context, spec *Spec) (*ssh.Client, error) {
	client, err := dial(
		spec.ip,
		spec.Settings.Username,
		spec.Settings.Password,
	)
	if err == nil {
		return client, nil
	}

	ctx, cancel := context.WithTimeout(ctx, networkTimeout)
	defer cancel()
	for i := 0; ; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		logger.FromContext(ctx).
			WithField("ip", spec.ip).
			WithField("id", spec.Name).
			WithField("attempt", i).
			Trace("dialing the vm")
		client, err = dial(
			spec.ip,
			spec.Settings.Username,
			spec.Settings.Password,
		)
		if err == nil {
			return client, nil
		}
		logger.FromContext(ctx).
			WithError(err).
			WithField("ip", spec.ip).
			WithField("id", spec.Name).
			WithField("attempt", i).
			Trace("failed to re-dial vm")

		if client != nil {
			client.Close()
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second * 10):
		}
	}
}

// helper function configures and dials the ssh server.
func dial(server, username, password string) (*ssh.Client, error) {
	return ssh.Dial("tcp", server, &ssh.ClientConfig{
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),

		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
	})
}

// helper function writes the file to the remote server and then
// configures the file permissions.
func upload(client *sftp.Client, path string, data []byte, mode uint32) error {
	f, err := client.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return err
	}
	err = f.Chmod(os.FileMode(mode))
	if err != nil {
		return err
	}
	return nil
}

// helper function creates the folder on the remote server and
// then configures the folder permissions.
func mkdir(client *sftp.Client, path string, mode uint32) error {
	err := client.MkdirAll(path)
	if err != nil {
		return err
	}
	return client.Chmod(path, os.FileMode(mode))
}
