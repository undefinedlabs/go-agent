package scopeagent // import "go.undefinedlabs.com/scopeagent"

import (
	"go.undefinedlabs.com/scopeagent/agent"
	"go.undefinedlabs.com/scopeagent/instrumentation"
	scopetesting "go.undefinedlabs.com/scopeagent/instrumentation/testing"
	"testing"
)

var (
	GlobalAgent *agent.Agent
)

func init() {
	defaultAgent, err := agent.NewAgent()
	if err == nil {
		return
	}

	GlobalAgent = defaultAgent

	if agent.GetBoolEnv("SCOPE_SET_GLOBAL_TRACER", true) {
		GlobalAgent.SetAsGlobalTracer()
	}

	if agent.GetBoolEnv("SCOPE_AUTO_INSTRUMENT", true) {
		if err := instrumentation.PatchAll(); err != nil {
			panic(err)
		}
	}
}

func Run(m *testing.M) int {
	return GlobalAgent.Run(m)
}

func StartTest(t *testing.T, opts ...scopetesting.Option) *scopetesting.Test {
	opts = append(opts, scopetesting.WithOnPanicHandler(func(test *scopetesting.Test) {
		_ = GlobalAgent.Flush()
		GlobalAgent.PrintReport()
	}))
	test := scopetesting.StartTest(t, opts...)
	return test
}
