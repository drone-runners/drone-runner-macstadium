// Copyright 2020 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package orka

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/drone/runner-go/logger"

	"github.com/hashicorp/go-multierror"
)

// ErrInsufficientCPU is returned when the cluster has insufficient CPU
// to deploy the virtual machine.
var ErrInsufficientCPU = errors.New("No available nodes with sufficient CPU.")

// Client provides a macstadium client.
type Client struct {
	Client   *http.Client
	Dumper   logger.Dumper
	Endpoint string
	Token    string
}

// Create creates a deployment.
func (c *Client) Create(ctx context.Context, config *Config) (*Response, error) {
	in := map[string]interface{}{
		"orka_vm_name":    config.Name,
		"orka_base_image": config.Image,
		"orka_image":      config.Name,
		"orka_cpu_core":   config.CPU,
		"vcpu_count":      config.VCPU,
	}
	uri := fmt.Sprintf("%s/resources/vm/create", c.Endpoint)
	out := new(Response)
	err := c.do("POST", uri, &in, out)
	if err != nil {
		return nil, err
	}
	return out, getErrors(*out)
}

// Deploy deploys a virtual machine.
func (c *Client) Deploy(ctx context.Context, name string) (*DeployResponse, error) {
	in := map[string]string{"orka_vm_name": name}
	uri := fmt.Sprintf("%s/resources/vm/deploy", c.Endpoint)
	out := new(DeployResponse)
	err := c.do("POST", uri, &in, out)
	if err != nil {
		return nil, err
	}
	return out, getErrors(out.Response)
}

// Delete deletes a deployment and deployment configuration.
func (c *Client) Delete(ctx context.Context, name string) (*Response, error) {
	in := map[string]string{"orka_vm_name": name}
	uri := fmt.Sprintf("%s/resources/vm/purge", c.Endpoint)
	out := new(Response)
	err := c.do("DELETE", uri, &in, out)
	if err != nil {
		return nil, err
	}
	return out, getErrors(*out)
}

// Check checks the virtual machine status.
func (c *Client) Check(ctx context.Context, name string) (*StatusResponse, error) {
	uri := fmt.Sprintf("%s/resources/vm/status/%s", c.Endpoint, name)
	out := new(StatusResponse)
	err := c.do("GET", uri, nil, out)
	if err != nil {
		return nil, err
	}
	return out, getErrors(out.Response)
}

// CheckToken checks the token status
func (c *Client) CheckToken(ctx context.Context) (*TokenResponse, error) {
	uri := fmt.Sprintf("%s/token", c.Endpoint)
	out := new(TokenResponse)
	err := c.do("GET", uri, nil, out)
	if err != nil {
		return nil, err
	}
	return out, getErrors(out.Response)
}

// do makes an http.Request to the target endpoint.
func (c *Client) do(method, endpoint string, in, out interface{}) error {
	req, err := http.NewRequest(method, endpoint, nil)
	if err != nil {
		return err
	}

	if in != nil {
		dec, _ := json.Marshal(in)
		buf := bytes.NewBuffer(dec)
		req.Body = ioutil.NopCloser(buf)
		req.ContentLength = int64(len(dec))
		req.Header.Set("Content-Length", fmt.Sprint(len(dec)))
		req.Header.Set("Content-Type", "application/json")
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.Token))

	if c.Dumper != nil {
		c.Dumper.DumpRequest(req)
	}

	res, err := c.client().Do(req)
	if res != nil && res.Body != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return err
	}

	if c.Dumper != nil {
		c.Dumper.DumpResponse(res)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, out)
}

func (c *Client) client() *http.Client {
	if c.Client == nil {
		return http.DefaultClient
	}
	return c.Client
}

func getErrors(r Response) error {
	var result error
	for _, err := range r.Errors {
		switch err.Message {
		case ErrInsufficientCPU.Error():
			return ErrInsufficientCPU
		}
		result = multierror.Append(result, err)
	}
	return result
}
