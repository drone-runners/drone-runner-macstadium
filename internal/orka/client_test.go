// Copyright 2020 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package orka

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/h2non/gock"
)

func TestDeploy(t *testing.T) {
	defer gock.Off()

	gock.New("http://10.221.188.100").
		Post("resources/vm/deploy").
		Reply(200).
		Type("application/json").
		File("testdata/deploy.json")

	client := &Client{
		Endpoint: "http://10.221.188.100",
		Token:    "token",
	}
	got, err := client.Deploy(context.Background(), "test")
	if err != nil {
		t.Error(err)
	}

	want := new(DeployResponse)
	raw, _ := ioutil.ReadFile("testdata/deploy.json.golden")
	json.Unmarshal(raw, &want)

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("Unexpected Results")
		t.Log(diff)
	}

	if !gock.IsDone() {
		t.Errorf("Pending mocks")
	}
}

func TestDeployError(t *testing.T) {
	defer gock.Off()

	gock.New("http://10.221.188.100").
		Post("resources/vm/deploy").
		Reply(200).
		Type("application/json").
		File("testdata/deploy_error.json")

	client := &Client{
		Endpoint: "http://10.221.188.100",
		Token:    "token",
	}
	_, err := client.Deploy(context.Background(), "test")
	if err == nil {
		t.Errorf("Expect deployment error")
	}

	if !gock.IsDone() {
		t.Errorf("Pending mocks")
	}
}

func dump(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}
