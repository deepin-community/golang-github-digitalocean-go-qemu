// Copyright 2016 The go-qemu Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package qemu

import (
	"errors"
	"io"
	"testing"

	"github.com/digitalocean/go-qemu/qmp"
)

func TestCancelJob(t *testing.T) {
	const device = "drive-virtio-disk0"
	d, done := testDomain(t, func(cmd qmp.Command) (interface{}, error) {
		if want, got := "block-job-cancel", cmd.Execute; want != got {
			t.Fatalf("unexpected QMP command:\n- want: %q\n-  got: %q",
				want, got)
		}

		args, _ := cmd.Args.(map[string]interface{})
		if want, got := device, args["device"]; want != got {
			t.Fatalf("unexpected device:\n- want: %q\n-  got: %q",
				want, got)
		}

		return success{}, nil
	})
	defer done()

	job := BlockJob{Device: device}
	err := job.Cancel(d, defaultTestTimeout)
	if err != nil {
		t.Error(err)
	}
}

func TestCommit(t *testing.T) {
	const (
		device  = "inserted[node-name] before commit"
		overlay = "/tmp/foo.img"
		jobID   = "made-up-job-id-for-the-commit"
	)
	d, done := testDomain(t, func(cmd qmp.Command) (interface{}, error) {
		if want, got := "block-commit", cmd.Execute; want != got {
			t.Fatalf("unexpected QMP command:\n- want: %q\n-  got: %q",
				want, got)
		}

		args, _ := cmd.Args.(map[string]interface{})
		if want, got := device, args["device"]; want != got {
			t.Fatalf("unexpected device:\n- want: %q\n-  got: %q",
				want, got)
		}
		if want, got := overlay, args["top"]; want != got {
			t.Fatalf("unexpected device:\n- want: %q\n-  got: %q",
				want, got)
		}
		if want, got := jobID, args["job-id"]; want != got {
			t.Fatalf("unexpected job-id:\n- want: %q\n-  got: %q",
				want, got)
		}

		return success{}, nil
	})
	defer done()

	disk := BlockDevice{}
	disk.Inserted.NodeName = device
	err := disk.Commit(d, overlay, jobID, defaultTestTimeout)
	if err != nil {
		t.Error(err)
	}
}

func TestCommitActiveBlockJob(t *testing.T) {
	d, done := testDomain(t, func(_ qmp.Command) (interface{}, error) {
		return failure{
			Error: map[string]string{
				"class": "GenericError",
			},
		}, errors.New("fail")
	})
	defer done()

	disk := BlockDevice{}
	err := disk.Commit(d, "/tmp/foo", "job-id", defaultTestTimeout)
	if err == nil {
		t.Errorf("expected blockcommit with active blockjob to fail")
	}
}

func TestCommitBlockJobError(t *testing.T) {
	d, done := testDomain(t, func(_ qmp.Command) (interface{}, error) {
		return success{}, nil
	}, withEventErrors)
	defer done()

	disk := BlockDevice{}
	err := disk.Commit(d, "/tmp/foo", "job-id", defaultTestTimeout)
	if err == nil {
		t.Error("expected block job error to cause failure")
	} else if errors.Is(err, io.EOF) {
		t.Error("didn't expect the event stream to close")
	}
}

func TestCommitTimeout(t *testing.T) {
	d, done := testDomain(t, func(_ qmp.Command) (interface{}, error) {
		return success{}, nil
	}, withEventTimeout)
	defer done()

	disk := BlockDevice{Device: "test"}
	err := disk.Commit(d, "/tmp/foo", "job-id", 0)
	if err == nil {
		t.Error("expected timeout")
	}
}

func TestJobComplete(t *testing.T) {
	const device = "drive-virtio-disk0"
	d, done := testDomain(t, func(cmd qmp.Command) (interface{}, error) {
		if want, got := "block-job-complete", cmd.Execute; want != got {
			t.Fatalf("unexpected QMP command:\n- want: %q\n-  got: %q",
				want, got)
		}

		args, _ := cmd.Args.(map[string]interface{})
		if want, got := device, args["device"]; want != got {
			t.Fatalf("unexpected device:\n- want: %q\n-  got: %q",
				want, got)
		}

		return success{}, nil
	})
	defer done()

	job := BlockJob{Device: device}
	err := job.Complete(d, defaultTestTimeout)
	if err != nil {
		t.Error(err)
	}
}

func TestJobCompleteEventError(t *testing.T) {
	d, done := testDomain(t, func(_ qmp.Command) (interface{}, error) {
		return success{}, nil
	}, withEventErrors)
	defer done()

	job := BlockJob{Device: "test"}
	err := job.Complete(d, defaultTestTimeout)
	if err == nil {
		t.Error("expected block job error to cause failure")
	}
}

func TestJobCompleteTimeout(t *testing.T) {
	d, done := testDomain(t, func(_ qmp.Command) (interface{}, error) {
		return success{}, nil
	}, withEventTimeout)
	defer done()

	job := BlockJob{Device: "test"}
	err := job.Complete(d, 0)
	if err == nil {
		t.Error("expected timeout")
	}
}

func TestMirror(t *testing.T) {
	const (
		device = "drive-virtio-disk0"
		dest   = "/tmp/foo.img"
	)
	d, done := testDomain(t, func(cmd qmp.Command) (interface{}, error) {
		if want, got := "drive-mirror", cmd.Execute; want != got {
			t.Fatalf("unexpected QMP command:\n- want: %q\n-  got: %q",
				want, got)
		}

		args, _ := cmd.Args.(map[string]interface{})
		if want, got := device, args["device"]; want != got {
			t.Fatalf("unexpected device:\n- want: %q\n-  got: %q",
				want, got)
		}

		if want, got := dest, args["target"]; want != got {
			t.Fatalf("unexpected target:\n- want: %q\n-  got: %q",
				want, got)
		}

		return success{}, nil
	})
	defer done()

	disk := BlockDevice{Device: device}
	err := disk.Mirror(d, dest, defaultTestTimeout)
	if err != nil {
		t.Error(err)
	}
}

func TestMirrorRelativePath(t *testing.T) {
	const (
		device = "drive-virtio-disk0"
		dest   = "relative-path.img"
	)
	d, done := testDomain(t, func(_ qmp.Command) (interface{}, error) {
		return success{}, nil
	})
	defer done()

	disk := BlockDevice{Device: device}
	err := disk.Mirror(d, dest, defaultTestTimeout)
	if err == nil {
		t.Errorf("expected relative path %q to fail", dest)
	}
}

func TestSnapshot(t *testing.T) {
	const (
		device   = "drive-virtio-disk0"
		overlay  = "/tmp/foo.img"
		nodeName = "my-node"
	)
	d, done := testDomain(t, func(cmd qmp.Command) (interface{}, error) {
		if want, got := "blockdev-snapshot-sync", cmd.Execute; want != got {
			t.Fatalf("unexpected QMP command:\n- want: %q\n-  got: %q",
				want, got)
		}

		args, _ := cmd.Args.(map[string]interface{})
		if want, got := device, args["node-name"]; want != got {
			t.Fatalf("unexpected device:\n- want: %q\n-  got: %q",
				want, got)
		}

		if want, got := overlay, args["snapshot-file"]; want != got {
			t.Fatalf("unexpected target:\n- want: %q\n-  got: %q",
				want, got)
		}

		if want, got := nodeName, args["snapshot-node-name"]; want != got {
			t.Fatalf("unexpected target:\n- want: %q\n-  got: %q",
				want, got)
		}

		return success{}, nil
	})
	defer done()

	disk := BlockDevice{}
	disk.Inserted.NodeName = device
	err := disk.Snapshot(d, overlay, "my-node")
	if err != nil {
		t.Error(err)
	}
}
