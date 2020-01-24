package testing

import (
	"flag"
	"os"
	"reflect"
	"runtime"
	"testing"

	"github.com/undefinedlabs/go-mpatch"

	"go.undefinedlabs.com/scopeagent/instrumentation"
	"go.undefinedlabs.com/scopeagent/reflection"
)

var (
	parallel int
)

// Initialize the testing instrumentation
func Init(m *testing.M) {
	flag.Parse()
	fPtr := flag.Lookup("test.parallel")
	if fPtr != nil {
		parallel = (*fPtr).Value.(flag.Getter).Get().(int)
		instrumentation.Logger().Println("parallel flag set to:", parallel)
	}

	if tPointer, err := reflection.GetFieldPointerOf(m, "tests"); err == nil {
		intTests := (*[]testing.InternalTest)(tPointer)
		tests := make([]testing.InternalTest, 0)
		for _, test := range *intTests {
			funcValue := test.F
			funcPointer := reflect.ValueOf(funcValue).Pointer()
			tests = append(tests, testing.InternalTest{
				Name: test.Name,
				F: func(t *testing.T) { // Creating a new test function as an indirection of the original test
					addAutoInstrumentedTest(t)
					tStruct := StartTestFromCaller(t, funcPointer)
					defer tStruct.end()
					funcValue(t)
				},
			})
		}
		// Replace internal tests with new test indirection
		*intTests = tests
	}
	if bPointer, err := reflection.GetFieldPointerOf(m, "benchmarks"); err == nil {
		intBenchmarks := (*[]testing.InternalBenchmark)(bPointer)
		var benchmarks []testing.InternalBenchmark
		for _, benchmark := range *intBenchmarks {
			funcValue := benchmark.F
			funcPointer := reflect.ValueOf(funcValue).Pointer()
			benchmarks = append(benchmarks, testing.InternalBenchmark{
				Name: benchmark.Name,
				F: func(b *testing.B) { // Indirection of the original benchmark
					startBenchmark(b, funcPointer, funcValue)
				},
			})
		}
		*intBenchmarks = benchmarks
	}

	if envDMPatch, set := os.LookupEnv("SCOPE_DISABLE_MONKEY_PATCHING"); !set || envDMPatch == "" {
		// We monkey patch the `testing.T.Run()` func to auto instrument sub tests
		var t *testing.T
		tType := reflect.TypeOf(t)
		if tRunMethod, ok := tType.MethodByName("Run"); ok {
			var runPatch *mpatch.Patch
			var err error
			runPatch, err = mpatch.PatchMethodByReflect(tRunMethod, func(t *testing.T, name string, f func(t *testing.T)) bool {
				pc, _, _, _ := runtime.Caller(1)
				logOnError(runPatch.Unpatch())
				defer runPatch.Patch()
				return t.Run(name, func(childT *testing.T) {
					_ = runPatch.Patch()
					addAutoInstrumentedTest(childT)
					childTest := StartTestFromCaller(childT, pc)
					defer childTest.end()
					f(childT)
				})
			})
			logOnError(err)
		}
	}
}
