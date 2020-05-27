/*
 Copyright 2020 Qiniu Cloud (qiniu.com)

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package cover

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testCoverage() (c *Coverage) {
	return &Coverage{FileName: "fake-coverage", NCoveredStmts: 200, NAllStmts: 300}
}

func TestCoverageRatio(t *testing.T) {
	c := testCoverage()
	actualRatio, _ := c.Ratio()
	assert.Equal(t, float32(c.NCoveredStmts)/float32(c.NAllStmts), actualRatio)
}

func TestRatioErr(t *testing.T) {
	c := &Coverage{FileName: "fake-coverage", NCoveredStmts: 200, NAllStmts: 0}
	_, err := c.Ratio()
	assert.NotNil(t, err)
}

func TestPercentageNA(t *testing.T) {
	c := &Coverage{FileName: "fake-coverage", NCoveredStmts: 200, NAllStmts: 0}
	assert.Equal(t, "N/A", c.Percentage())
}

func TestGenLocalCoverDiffReport(t *testing.T) {
	//coverage increase
	newList := &CoverageList{Groups: []Coverage{{FileName: "fake-coverage", NCoveredStmts: 15, NAllStmts: 20}}}
	baseList := &CoverageList{Groups: []Coverage{{FileName: "fake-coverage", NCoveredStmts: 10, NAllStmts: 20}}}
	rows := GenLocalCoverDiffReport(newList, baseList)
	assert.Equal(t, 1, len(rows))
	assert.Equal(t, []string{"fake-coverage", "50.0%", "75.0%", "25.0%"}, rows[0])

	//coverage decrease
	baseList = &CoverageList{Groups: []Coverage{{FileName: "fake-coverage", NCoveredStmts: 20, NAllStmts: 20}}}
	rows = GenLocalCoverDiffReport(newList, baseList)
	assert.Equal(t, []string{"fake-coverage", "100.0%", "75.0%", "-25.0%"}, rows[0])

	//diff file
	baseList = &CoverageList{Groups: []Coverage{{FileName: "fake-coverage-v1", NCoveredStmts: 10, NAllStmts: 20}}}
	rows = GenLocalCoverDiffReport(newList, baseList)
	assert.Equal(t, []string{"fake-coverage", "None", "75.0%", "75.0%"}, rows[0])
}

func TestCovList(t *testing.T) {
	fileName := "qiniu.com/kodo/apiserver/server/main.go"

	// percentage is 100%
	p := strings.NewReader("mode: atomic\n" +
		fileName + ":32.49,33.13 1 30\n")
	covL, err := CovList(p)
	covF := covL.Map()[fileName]
	assert.Nil(t, err)
	assert.Equal(t, "100.0%", covF.Percentage())

	// percentage is 50%
	p = strings.NewReader("mode: atomic\n" +
		fileName + ":32.49,33.13 1 30\n" +
		fileName + ":42.49,43.13 1 0\n")
	covL, err = CovList(p)
	covF = covL.Map()[fileName]
	assert.Nil(t, err)
	assert.Equal(t, "50.0%", covF.Percentage())

	// two files
	fileName1 := "qiniu.com/kodo/apiserver/server/svr.go"
	p = strings.NewReader("mode: atomic\n" +
		fileName + ":32.49,33.13 1 30\n" +
		fileName1 + ":42.49,43.13 1 0\n")
	covL, err = CovList(p)
	covF = covL.Map()[fileName]
	covF1 := covL.Map()[fileName1]
	assert.Nil(t, err)
	assert.Equal(t, "100.0%", covF.Percentage())
	assert.Equal(t, "0.0%", covF1.Percentage())
}

func TestBuildCoverCmd(t *testing.T) {
	var testCases = []struct {
		name      string
		file      string
		coverVar  *FileVar
		pkg       *Package
		mode      string
		newgopath string
		expectCmd *exec.Cmd
	}{
		{
			name: "normal",
			file: "c.go",
			coverVar: &FileVar{
				File: "example/b/c/c.go",
				Var:  "GoCover_0_643131623532653536333031",
			},
			pkg: &Package{
				Dir: "/go/src/goc/cmd/example-project/b/c",
			},
			mode:      "count",
			newgopath: "",
			expectCmd: &exec.Cmd{
				Path: lookCmdPath("go"),
				Args: []string{"go", "tool", "cover", "-mode", "count", "-var", "GoCover_0_643131623532653536333031", "-o",
					"/go/src/goc/cmd/example-project/b/c/c.go", "/go/src/goc/cmd/example-project/b/c/c.go"},
			},
		},
		{
			name: "normal with gopath",
			file: "c.go",
			coverVar: &FileVar{
				File: "example/b/c/c.go",
				Var:  "GoCover_0_643131623532653536333031",
			},
			pkg: &Package{
				Dir: "/go/src/goc/cmd/example-project/b/c",
			},
			mode:      "set",
			newgopath: "/go/src/goc",
			expectCmd: &exec.Cmd{
				Path: lookCmdPath("go"),
				Args: []string{"go", "tool", "cover", "-mode", "set", "-var", "GoCover_0_643131623532653536333031", "-o",
					"/go/src/goc/cmd/example-project/b/c/c.go", "/go/src/goc/cmd/example-project/b/c/c.go"},
				Env: append(os.Environ(), fmt.Sprintf("GOPATH=%v", "/go/src/goc")),
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			cmd := buildCoverCmd(testCase.file, testCase.coverVar, testCase.pkg, testCase.mode, testCase.newgopath)
			if !reflect.DeepEqual(cmd, testCase.expectCmd) {
				t.Errorf("generated incorrect commands:\nGot: %#v\nExpected:%#v", cmd, testCase.expectCmd)
			}
		})
	}

}

func lookCmdPath(name string) string {
	if filepath.Base(name) == name {
		if lp, err := exec.LookPath(name); err != nil {
			log.Fatalf("find exec %s err: %v", name, err)
		} else {
			return lp
		}
	}
	return ""
}

func TestDeclareCoverVars(t *testing.T) {
	var testCases = []struct {
		name           string
		pkg            *Package
		expectCoverVar map[string]*FileVar
	}{
		{
			name: "normal",
			pkg: &Package{
				Dir:        "/go/src/goc/cmd/example-project/b/c",
				GoFiles:    []string{"c.go"},
				ImportPath: "example/b/c",
			},
			expectCoverVar: map[string]*FileVar{
				"c.go": {File: "example/b/c/c.go", Var: "GoCover_0_643131623532653536333031"},
			},
		},
		{
			name: "more go files",
			pkg: &Package{
				Dir:        "/go/src/goc/cmd/example-project/a/b",
				GoFiles:    []string{"printf.go", "printf1.go"},
				ImportPath: "example/a/b",
			},
			expectCoverVar: map[string]*FileVar{
				"printf.go":  {File: "example/a/b/printf.go", Var: "GoCover_0_326535623364613565313464"},
				"printf1.go": {File: "example/a/b/printf1.go", Var: "GoCover_1_326535623364613565313464"},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			coverVar := declareCoverVars(testCase.pkg)
			if !reflect.DeepEqual(coverVar, testCase.expectCoverVar) {
				t.Errorf("generated incorrect cover vars:\nGot: %#v\nExpected:%#v", coverVar, testCase.expectCoverVar)
			}
		})
	}

}
