// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

package config

import (
	"testing"

	"github.com/derailed/k9s/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomView_getVS(t *testing.T) {
	uu := map[string]struct {
		cv           *CustomView
		gvr, ns, ctx string
		e            *ViewSetting
	}{
		"empty": {},

		"miss": {
			gvr: "zorg",
		},

		"gvr": {
			gvr: client.PodGVR.String(),
			e: &ViewSetting{
				Columns: []string{"NAMESPACE", "NAME", "AGE", "IP"},
			},
		},

		"gvr+ns": {
			gvr: client.PodGVR.String(),
			ns:  "default",
			e: &ViewSetting{
				Columns: []string{"NAME", "IP", "AGE"},
			},
		},

		"rx": {
			gvr: client.PodGVR.String(),
			ns:  "ns-fred",
			e: &ViewSetting{
				Columns: []string{"AGE", "NAME", "IP"},
			},
		},

		"alias": {
			gvr: "bozo",
			e: &ViewSetting{
				Columns: []string{"DUH", "BLAH", "BLEE"},
			},
		},

		"toast-no-ns": {
			gvr: client.PodGVR.String(),
			ns:  "zorg",
			e: &ViewSetting{
				Columns: []string{"NAMESPACE", "NAME", "AGE", "IP"},
			},
		},

		"toast-no-res": {
			gvr: client.SvcGVR.String(),
			ns:  "zorg",
			e:   nil,
		},

		"context-match": {
			gvr: client.SvcGVR.String(),
			ctx: "prod-cluster",
			e: &ViewSetting{
				Columns: []string{"NAME", "TYPE", "CLUSTER-IP"},
			},
		},

		"context-no-match": {
			gvr: client.SvcGVR.String(),
			ctx: "dev-cluster",
			e:   nil,
		},

		"context-fallback": {
			gvr: client.PodGVR.String(),
			ctx: "any-cluster",
			e: &ViewSetting{
				Columns: []string{"NAMESPACE", "NAME", "AGE", "IP"},
			},
		},

		"ns+context": {
			gvr: client.PodGVR.String(),
			ns:  "kube-system",
			ctx: "production",
			e: &ViewSetting{
				Columns: []string{"NAME", "NODE", "STATUS", "AGE"},
			},
		},

		"ns+context-fallback-to-context": {
			gvr: client.PodGVR.String(),
			ns:  "other-namespace",
			ctx: "production",
			e: &ViewSetting{
				Columns: []string{"NAMESPACE", "NAME", "READY", "STATUS"},
			},
		},

		"ns+context-fallback-to-ns": {
			gvr: client.PodGVR.String(),
			ns:  "default",
			ctx: "other-context",
			e: &ViewSetting{
				Columns: []string{"NAME", "IP", "AGE"},
			},
		},
	}

	v := NewCustomView()
	require.NoError(t, v.Load("testdata/views/views.yaml"))
	for k, u := range uu {
		t.Run(k, func(t *testing.T) {
			assert.Equal(t, u.e, v.getVS(u.gvr, u.ns, u.ctx))
		})
	}
}
