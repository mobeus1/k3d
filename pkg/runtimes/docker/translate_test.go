/*
Copyright © 2020-2022 The k3d Author(s)

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package docker

import (
	"os"
	"strconv"
	"testing"

	"github.com/go-test/deep"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	k3d "github.com/rancher/k3d/v5/pkg/types"
	"github.com/rancher/k3d/v5/pkg/types/fixes"
)

func TestTranslateNodeToContainer(t *testing.T) {

	inputNode := &k3d.Node{
		Name:    "test",
		Role:    k3d.ServerRole,
		Image:   "rancher/k3s:v0.9.0",
		Volumes: []string{"/test:/tmp/test"},
		Env:     []string{"TEST_KEY_1=TEST_VAL_1"},
		Cmd:     []string{"server", "--https-listen-port=6443"},
		Args:    []string{"--some-boolflag"},
		Ports: nat.PortMap{
			"6443/tcp": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: "6443",
				},
			},
		},
		Restart:       true,
		RuntimeLabels: map[string]string{k3d.LabelRole: string(k3d.ServerRole), "test_key_1": "test_val_1"},
		Networks:      []string{"mynet"},
	}

	init := true
	if disableInit, err := strconv.ParseBool(os.Getenv(k3d.K3dEnvDebugDisableDockerInit)); err == nil && disableInit {
		init = false
	}

	expectedRepresentation := &NodeInDocker{
		ContainerConfig: container.Config{
			Hostname: "test",
			Image:    "rancher/k3s:v0.9.0",
			Env:      []string{"TEST_KEY_1=TEST_VAL_1"},
			Cmd:      []string{"server", "--https-listen-port=6443", "--some-boolflag"},
			Labels:   map[string]string{k3d.LabelRole: string(k3d.ServerRole), "test_key_1": "test_val_1"},
			ExposedPorts: nat.PortSet{
				"6443/tcp": struct{}{},
			},
		},
		HostConfig: container.HostConfig{
			Binds: []string{"/test:/tmp/test"},
			RestartPolicy: container.RestartPolicy{
				Name: "unless-stopped",
			},
			Init:       &init,
			Privileged: true,
			Tmpfs:      map[string]string{"/run": "", "/var/run": ""},
			PortBindings: nat.PortMap{
				"6443/tcp": {
					{
						HostIP:   "0.0.0.0",
						HostPort: "6443",
					},
				},
			},
		},
		NetworkingConfig: network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				"mynet": {},
			},
		},
	}

	// TODO: // FIXME: FixCgroupV2 - to be removed when fixed upstream
	if fixes.FixEnabledAny() {
		expectedRepresentation.ContainerConfig.Entrypoint = []string{"/bin/k3d-entrypoint.sh"}
	}

	actualRepresentation, err := TranslateNodeToContainer(inputNode)
	if err != nil {
		t.Error(err)
	}

	if diff := deep.Equal(actualRepresentation, expectedRepresentation); diff != nil {
		t.Errorf("Actual representation\n%+v\ndoes not match expected representation\n%+v\nDiff:\n%+v", actualRepresentation, expectedRepresentation, diff)
	}

}
