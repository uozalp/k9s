// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

package config

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/derailed/k9s/internal/client"
	"github.com/derailed/k9s/internal/config/data"
	"github.com/derailed/k9s/internal/config/json"
	"github.com/derailed/k9s/internal/slogs"
	"gopkg.in/yaml.v3"
)

// ViewConfigListener represents a view config listener.
type ViewConfigListener interface {
	// ViewSettingsChanged notifies listener the view configuration changed.
	ViewSettingsChanged(*ViewSetting)

	// GetNamespace return the view namespace
	GetNamespace() string

	// GetContextName return the view context name
	GetContextName() string
}

// ViewSetting represents a view configuration.
type ViewSetting struct {
	Columns    []string `yaml:"columns"`
	SortColumn string   `yaml:"sortColumn"`
	Context    string   `yaml:"context,omitempty"`
}

func (v *ViewSetting) HasCols() bool {
	return len(v.Columns) > 0
}

func (v *ViewSetting) IsBlank() bool {
	return v == nil || (len(v.Columns) == 0 && v.SortColumn == "")
}

func (v *ViewSetting) SortCol() (name string, asc bool, err error) {
	if v == nil || v.SortColumn == "" {
		return "", false, fmt.Errorf("no sort column specified")
	}
	tt := strings.Split(v.SortColumn, ":")
	if len(tt) < 2 {
		return "", false, fmt.Errorf("invalid sort column spec: %q. must be col-name:asc|desc", v.SortColumn)
	}

	return tt[0], tt[1] == "asc", nil
}

// Equals checks if two view settings are equal.
func (v *ViewSetting) Equals(vs *ViewSetting) bool {
	if v == nil && vs == nil {
		return true
	}
	if v == nil || vs == nil {
		return false
	}

	if c := slices.Compare(v.Columns, vs.Columns); c != 0 {
		return false
	}

	if cmp.Compare(v.Context, vs.Context) != 0 {
		return false
	}

	return cmp.Compare(v.SortColumn, vs.SortColumn) == 0
}

// CustomView represents a collection of view customization.
type CustomView struct {
	Views     map[string]ViewSetting `yaml:"views"`
	listeners map[string]ViewConfigListener
}

// NewCustomView returns a views configuration.
func NewCustomView() *CustomView {
	return &CustomView{
		Views:     make(map[string]ViewSetting),
		listeners: make(map[string]ViewConfigListener),
	}
}

// Reset clears out configurations.
func (v *CustomView) Reset() {
	for k := range v.Views {
		delete(v.Views, k)
	}
}

// Load loads view configurations.
func (v *CustomView) Load(path string) error {
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	bb, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := data.JSONValidator.Validate(json.ViewsSchema, bb); err != nil {
		slog.Warn("Validation failed. Please update your config and restart!",
			slogs.Path, path,
			slogs.Error, err,
		)
	}
	var in CustomView
	if err := yaml.Unmarshal(bb, &in); err != nil {
		return err
	}
	v.Views = in.Views
	v.fireConfigChanged()

	return nil
}

// AddListeners registers a new listener for various commands.
func (v *CustomView) AddListeners(l ViewConfigListener, cmds ...string) {
	for _, cmd := range cmds {
		if cmd != "" {
			v.listeners[cmd] = l
		}
	}
	v.fireConfigChanged()
}

// AddListener registers a new listener.
func (v *CustomView) AddListener(cmd string, l ViewConfigListener) {
	v.listeners[cmd] = l
	v.fireConfigChanged()
}

// RemoveListener unregister a listener.
func (v *CustomView) RemoveListener(l ViewConfigListener) {
	for k, list := range v.listeners {
		if list == l {
			delete(v.listeners, k)
		}
	}
}

// GetVS retrieves view settings for a given GVR, namespace, and context.
func (v *CustomView) GetVS(gvr, ns, context string) *ViewSetting {
	return v.getVS(gvr, ns, context)
}

func (v *CustomView) fireConfigChanged() {
	cmds := slices.Collect(maps.Keys(v.listeners))
	slices.SortFunc(cmds, func(a, b string) int {
		switch {
		case strings.Contains(a, "/") && !strings.Contains(b, "/"):
			return 1
		case !strings.Contains(a, "/") && strings.Contains(b, "/"):
			return -1
		default:
			return strings.Compare(a, b)
		}
	})
	type tuple struct {
		cmd string
		vs  *ViewSetting
	}
	var victim tuple
	for _, cmd := range cmds {
		listener := v.listeners[cmd]
		if vs := v.getVS(cmd, listener.GetNamespace(), listener.GetContextName()); vs != nil {
			slog.Debug("Reloading custom view settings", slogs.Command, cmd)
			victim = tuple{cmd, vs}
			break
		}
		victim = tuple{cmd, nil}
	}
	if victim.cmd != "" {
		v.listeners[victim.cmd].ViewSettingsChanged(victim.vs)
	}
}

func (v *CustomView) getVS(gvr, ns, context string) *ViewSetting {
	if client.IsAllNamespaces(ns) {
		ns = client.NamespaceAll
	}
	k := gvr
	kk := slices.Collect(maps.Keys(v.Views))
	slices.SortFunc(kk, strings.Compare)
	slices.Reverse(kk)

	// matchContext checks if view setting context matches current context
	matchContext := func(vs *ViewSetting) bool {
		if vs.Context == "" {
			return true // No context filter, matches all
		}
		if context == "" {
			return true // No current context provided, accept any
		}
		return vs.Context == context
	}

	// Check for most specific patterns first (highest priority)
	if context != "" {
		// 1. Namespace + Context pattern (most specific: "v1/pods@default@context:production")
		if ns != "" && ns != client.NamespaceAll {
			nsContextKey := gvr + "@" + ns + "@context:" + context
			if vs, ok := v.Views[nsContextKey]; ok {
				return &vs
			}
		}

		// 2. Context-only pattern (e.g., "v1/pods@context:production")
		contextKey := gvr + "@context:" + context
		if vs, ok := v.Views[contextKey]; ok {
			return &vs
		}
	}

	for _, key := range kk {
		// Skip keys with context suffixes if they don't match current context
		if strings.Contains(key, "@context:") {
			continue // Already handled above
		}

		if !strings.HasPrefix(key, gvr) && !strings.HasPrefix(gvr, key) {
			continue
		}

		switch {
		case strings.Contains(key, "@"):
			tt := strings.Split(key, "@")
			if len(tt) != 2 {
				break
			}
			nsk := gvr
			if ns != "" {
				nsk += "@" + ns
			}
			if rx, err := regexp.Compile(tt[1]); err == nil && rx.MatchString(nsk) {
				vs := v.Views[key]
				if matchContext(&vs) {
					return &vs
				}
			}
		case strings.HasPrefix(k, key):
			kk := strings.Fields(k)
			if len(kk) == 2 {
				if vs, ok := v.Views[kk[0]+"@"+kk[1]]; ok && matchContext(&vs) {
					return &vs
				}
				if key == kk[0] {
					vs := v.Views[key]
					if matchContext(&vs) {
						return &vs
					}
				}
			}
			fallthrough
		case key == k:
			vs := v.Views[key]
			if matchContext(&vs) {
				return &vs
			}
		}
	}

	return nil
}
