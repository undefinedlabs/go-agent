package ast

import (
	"go/ast"
	"go/parser"
	"go/token"
	"runtime"
	"strings"
	"sync"
)

var (
	methodCodes map[string]map[string]*MethodCodeBoundaries
	mutex       sync.Mutex
)

type MethodCodeBoundaries struct {
	Package string
	Name    string
	File    string
	Start   CodePos
	End     CodePos
}
type CodePos struct {
	Line   int
	Column int
}

// Gets the function source code boundaries from the caller method
func GetFuncSourceFromCaller(skip int) *MethodCodeBoundaries {
	pc, _, _, _ := runtime.Caller(skip + 1)
	return GetFuncSource(pc)
}

// Gets the function source code boundaries from a method
func GetFuncSource(pc uintptr) *MethodCodeBoundaries {
	mFunc := runtime.FuncForPC(pc)
	mFile, _ := mFunc.FileLine(pc)

	mutex.Lock()
	if methodCodes == nil {
		methodCodes = map[string]map[string]*MethodCodeBoundaries{}
	}
	if methodCodes[mFile] == nil {
		methodCodes[mFile] = map[string]*MethodCodeBoundaries{}

		fSet := token.NewFileSet()
		f, err := parser.ParseFile(fSet, mFile, nil, 0)
		if err != nil {
			return nil
		}

		packageName := f.Name.String()
		for _, decl := range f.Decls {
			if fDecl, ok := decl.(*ast.FuncDecl); ok {
				bPos := fDecl.Pos()
				if fDecl.Body != nil {
					bEnd := fDecl.Body.End()
					if bPos.IsValid() && bEnd.IsValid() {
						pos := fSet.PositionFor(bPos, true)
						end := fSet.PositionFor(bEnd, true)
						methodCode := MethodCodeBoundaries{
							Package: packageName,
							Name:    fDecl.Name.String(),
							File:    mFile,
							Start: CodePos{
								Line:   pos.Line,
								Column: pos.Column,
							},
							End: CodePos{
								Line:   end.Line,
								Column: end.Column,
							},
						}
						methodCodes[mFile][methodCode.Name] = &methodCode
					}
				}
			}
		}
	}
	mutex.Unlock()

	funcName := mFunc.Name()
	funcNameParts := strings.Split(funcName, ".")
	return methodCodes[mFile][funcNameParts[1]]
}
