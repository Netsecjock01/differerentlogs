// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build !integration
// +build !integration

package container

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/metricbeat/module/kubernetes/util"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const testFile = "../_meta/test/stats_summary.json"

func TestEventMapping(t *testing.T) {
	logger := logp.NewLogger("kubernetes.container")

	f, err := os.Open(testFile)
	assert.NoError(t, err, "cannot open test file "+testFile)

	body, err := ioutil.ReadAll(f)
	assert.NoError(t, err, "cannot read test file "+testFile)

	metricsRepo := util.NewMetricsRepo()

	nodeName := "gke-beats-default-pool-a5b33e2e-hdww"

	nodeMetrics := util.NewNodeMetrics()
	nodeMetrics.CoresAllocatable = util.NewFloat64Metric(2)
	nodeMetrics.MemoryAllocatable = util.NewFloat64Metric(146227200)
	addNodeMetric(metricsRepo, nodeName, nodeMetrics)

	namespace := "default"
	podName := "nginx-deployment-2303442956-pcqfc"
	podId := util.NewPodId(namespace, podName)
	containerName := "nginx"

	containerMetrics := util.NewContainerMetrics()
	containerMetrics.MemoryLimit = util.NewFloat64Metric(14622720)
	addContainerMetric(metricsRepo, nodeName, podId, containerName, containerMetrics)

	events, err := eventMapping(body, metricsRepo, logger)
	assert.NoError(t, err, "error mapping "+testFile)

	assert.Len(t, events, 1, "got wrong number of events")

	testCases := map[string]interface{}{
		"cpu.usage.core.ns":   43959424,
		"cpu.usage.nanocores": 11263994,

		"logs.available.bytes": int64(98727014400),
		"logs.capacity.bytes":  int64(101258067968),
		"logs.used.bytes":      28672,
		"logs.inodes.count":    6258720,
		"logs.inodes.free":     6120096,
		"logs.inodes.used":     138624,

		"memory.available.bytes":  0,
		"memory.usage.bytes":      1462272,
		"memory.rss.bytes":        1409024,
		"memory.workingset.bytes": 1454080,
		"memory.pagefaults":       841,
		"memory.majorpagefaults":  0,

		// calculated pct fields:
		"cpu.usage.node.pct":          0.005631997,
		"cpu.usage.limit.pct":         0.005631997,
		"memory.usage.node.pct":       0.01,
		"memory.usage.limit.pct":      0.1,
		"memory.workingset.limit.pct": 0.09943977591036414,

		"name": "nginx",

		"rootfs.available.bytes": int64(98727014400),
		"rootfs.capacity.bytes":  int64(101258067968),
		"rootfs.used.bytes":      61440,
		"rootfs.inodes.used":     21,
	}

	for k, v := range testCases {
		testValue(t, events[0], k, v)
	}

	containerEcsFields := ecsfields(events[0], logger)
	testEcs := map[string]interface{}{
		"cpu.usage":    0.005631997,
		"memory.usage": 0.01,
		"name":         "nginx",
	}
	for k, v := range testEcs {
		testValue(t, containerEcsFields, k, v)
	}
}

func testValue(t *testing.T, event mapstr.M, field string, value interface{}) {
	data, err := event.GetValue(field)
	assert.NoError(t, err, "Could not read field "+field)
	assert.EqualValues(t, data, value, "Wrong value for field "+field)
}

func addContainerMetric(metricsRepo *util.MetricsRepo, nodeName string, podId util.PodId, containerName string, metrics *util.ContainerMetrics) {
	nodeStore, _ := metricsRepo.AddNodeStore(nodeName)
	podStore, _ := nodeStore.AddPodStore(podId)
	containerStore, _ := podStore.AddContainerStore(containerName)
	containerStore.SetContainerMetrics(metrics)
}

func addNodeMetric(metricsRepo *util.MetricsRepo, nodeName string, nodeMetrics *util.NodeMetrics) {
	nodeStore, _ := metricsRepo.AddNodeStore(nodeName)
	nodeStore.SetNodeMetrics(nodeMetrics)
}
