package autoinstrument

import (
	"reflect"
	"sync"
	"testing"

	"github.com/undefinedlabs/go-mpatch"

	"go.undefinedlabs.com/scopeagent"
	"go.undefinedlabs.com/scopeagent/env"
	"go.undefinedlabs.com/scopeagent/instrumentation"
)

var (
	once sync.Once
)

func init() {
	once.Do(func() {
		if env.ScopeDisableMonkeyPatching.AsBool(false) {
			return
		}
		var m *testing.M
		var mRunMethod reflect.Method
		var ok bool
		mType := reflect.TypeOf(m)
		if mRunMethod, ok = mType.MethodByName("Run"); !ok {
			return
		}

		var runPatch *mpatch.Patch
		var err error
		runPatch, err = mpatch.PatchMethodByReflect(mRunMethod, func(m *testing.M) int {
			logOnError(runPatch.Unpatch())
			defer func() {
				logOnError(runPatch.Patch())
			}()
			return scopeagent.Run(m)
		})
		logOnError(err)
	})
}

func logOnError(err error) {
	if err != nil {
		instrumentation.Logger().Println(err)
	}
}
