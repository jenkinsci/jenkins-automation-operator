package base

import (
	"testing"

	"github.com/stretchr/testify/assert"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func TestValidatePlugins(t *testing.T) {
	baseReconcileLoop := New(nil, nil, logf.ZapLogger(false),
		nil, false, false)
	t.Run("happy", func(t *testing.T) {
		plugins := map[string][]string{
			"valid-plugin-name:1.0": {
				"valid-plugin-name:1.0",
			},
		}
		got := baseReconcileLoop.validatePlugins(plugins)
		assert.Equal(t, true, got)
	})
	t.Run("fail, no version in plugin name", func(t *testing.T) {
		plugins := map[string][]string{
			"invalid-plugin-name": {
				"invalid-plugin-name",
			},
		}
		got := baseReconcileLoop.validatePlugins(plugins)
		assert.Equal(t, false, got)
	})
	t.Run("fail, no version in root plugin name", func(t *testing.T) {
		plugins := map[string][]string{
			"invalid-plugin-name": {
				"invalid-plugin-name:1.0",
			},
		}
		got := baseReconcileLoop.validatePlugins(plugins)
		assert.Equal(t, false, got)
	})
	t.Run("fail, no version in plugin name", func(t *testing.T) {
		plugins := map[string][]string{
			"invalid-plugin-name:1.0": {
				"invalid-plugin-name",
			},
		}
		got := baseReconcileLoop.validatePlugins(plugins)
		assert.Equal(t, false, got)
	})
	t.Run("happy", func(t *testing.T) {
		plugins := map[string][]string{
			"valid-plugin-name:1.0": {
				"valid-plugin-name:1.0",
				"valid-plugin-name2:1.0",
			},
		}
		got := baseReconcileLoop.validatePlugins(plugins)
		assert.Equal(t, true, got)
	})
	t.Run("hapy", func(t *testing.T) {
		plugins := map[string][]string{
			"valid-plugin-name:1.0": {},
		}
		got := baseReconcileLoop.validatePlugins(plugins)
		assert.Equal(t, true, got)
	})

}
