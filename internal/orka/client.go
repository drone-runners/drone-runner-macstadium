// Copyright 2020 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package orka

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/drone/runner-go/logger"
)

// Client provides a macstadium client.
type Client struct {
	Client   *http.Client
	Dumper   logger.Dumper
	Endpoint string
	Token    string
}

// Create creates a deployment.
func (c *Client) Create(ctx context.Context, config *Config) (*CreateResponse, error) {
	in := map[string]interface{}{
		"orka_vm_name":    config.Name,
		"orka_base_image": config.Image,
		"orka_image":      config.Name,
		"orka_cpu_core":   config.CPU,
		"vcpu_count":      config.VCPU,
	}
	uri := fmt.Sprintf("%s/resources/vm/create", c.Endpoint)
	out := new(CreateResponse)
	err := c.Do("POST", uri, &in, out)
	return out, err
}

// Deploy deploys a virtual machine.
func (c *Client) Deploy(ctx context.Context, name string) (*DeployResponse, error) {
	in := map[string]string{"orka_vm_name": name}
	uri := fmt.Sprintf("%s/resources/vm/deploy", c.Endpoint)
	out := new(DeployResponse)
	err := c.Do("POST", uri, &in, out)
	return out, err
}

// Delete deletes a deployment and deployment configuration.
func (c *Client) Delete(ctx context.Context, name string) (*DeleteResponse, error) {
	in := map[string]string{"orka_vm_name": name}
	uri := fmt.Sprintf("%s/resources/vm/purge", c.Endpoint)
	out := new(DeleteResponse)
	err := c.Do("DELETE", uri, &in, out)
	return out, err
}

// Check checks the virtual machine status.
func (c *Client) Check(ctx context.Context, name string) (*StatusResponse, error) {
	uri := fmt.Sprintf("%s/resources/vm/status/%s", c.Endpoint, name)
	out := new(StatusResponse)
	err := c.Do("GET", uri, nil, out)
	return out, err
}

// CheckToken checks the token status
func (c *Client) CheckToken(ctx context.Context) (*TokenResponse, error) {
	uri := fmt.Sprintf("%s/token", c.Endpoint)
	out := new(TokenResponse)
	err := c.Do("GET", uri, nil, out)
	return out, err
}

// Do makes an http.Request to the target endpoint.
func (s *Client) Do(method, endpoint string, in, out interface{}) error {
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

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", s.Token))

	if s.Dumper != nil {
		s.Dumper.DumpRequest(req)
	}

	res, err := s.client().Do(req)
	if res != nil && res.Body != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return err
	}

	if s.Dumper != nil {
		s.Dumper.DumpResponse(res)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, out)
}

func (s *Client) client() *http.Client {
	if s.Client == nil {
		return http.DefaultClient
	}
	return s.Client
}
