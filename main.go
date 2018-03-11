package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	plist "github.com/DHowett/go-plist"
	"github.com/bitrise-io/go-utils/log"
)

func failf(f string, args ...interface{}) {
	log.Errorf(f, args...)
	os.Exit(1)
}

func main() {
	dirname := os.Getenv("BITRISE_SOURCE_DIR")

	xmls := []string{}
	err := filepath.Walk(dirname, getFilesByExt("xml", &xmls))
	if err != nil {
		log.Errorf("%s", err)
	}

	// Android match
	androidFiles, err := filterJUnitTestResults(&xmls)
	if err != nil {
		log.Errorf("%s", err)
	}

	testModel := model{BuildSlug: os.Getenv("BITRISE_BUILD_SLUG"), TestResults: []result{}}

	for _, path := range androidFiles {
		f, err := os.Open(path)
		if err != nil {
			failf("%s", err)
		}
		b, err := ioutil.ReadAll(f)
		if err != nil {
			failf("%s", err)
		}
		testModel.TestResults = append(testModel.TestResults, result{Path: path, Content: b})
	}

	derivedDataPath := getDerivedDataPath()

	plists := []string{}
	err = filepath.Walk(derivedDataPath, getFilesByExt("plist", &plists))
	if err != nil {
		log.Errorf("%s", err)
	}

	// iOS match
	xcodeFiles, err := filterXcodeTestResults(&plists)
	if err != nil {
		log.Errorf("%s", err)
	}

	for _, path := range xcodeFiles {
		f, err := os.Open(path)
		if err != nil {
			failf("%s", err)
		}

		decoder := plist.NewDecoder(f)
		intf := XCTests{}
		err = decoder.Decode(&intf)
		if err != nil {
			failf("%s", err)
		}

		intf = intf.CleanSubTests()

		for _, test := range intf.TestableSummaries {
			testData := TestData{TestCases: []TestCase{}}
			testData.Name = test.TestName

			for _, t := range test.Tests {
				for _, testCase := range t.SubTests {
					if len(testCase.FailureSummaries) > 0 {
						testData.Failures++
					}
					for _, summary := range testCase.FailureSummaries {
						testData.TestCases = append(testData.TestCases, TestCase{Time: testCase.Duration, Failure: &summary.Message, ClassName: fmt.Sprintf("%s:%d", strings.TrimPrefix(strings.TrimPrefix(summary.FileName, os.Getenv("BITRISE_SOURCE_DIR")), "/"), summary.LineNumber), Name: testCase.TestID})
					}
				}
			}

			b, err := xml.Marshal(testData)
			if err != nil {
				failf("%s", err)
			}

			testModel.TestResults = append(testModel.TestResults, result{Path: path, Content: b})
		}
		// b, err := json.MarshalIndent(intf, "", " ")
		// if err != nil {
		// 	failf("%s", err)
		// }
		// fmt.Println(string(b))

		// fmt.Println()
	}

	b, err := json.MarshalIndent(testModel, "", " ")
	if err != nil {
		failf("%s", err)
	}

	client := &http.Client{}
	request, err := http.NewRequest("POST", "https://frozen-brushlands-50401.herokuapp.com/results", bytes.NewReader(b))
	if err != nil {
		failf("%s", err)
	}
	request.Header.Add("content-type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		failf("%s", err)
	}

	b, err = ioutil.ReadAll(response.Body)
	if err != nil {
		failf("%s", err)
	}

	fmt.Println(string(b))

	fmt.Println()
	//fmt.Println(xcodeFiles)

	// "TestableSummaries" (testsuites)
	//   "Tests" (test cases)
	//     "Subtests" ..-> deepest "Subtests" (test cases)
	//       "TestIdentifier"
	//       "TestStatus"
	//       "Duration"
	//       "FailureSummaries"
	//         "FileName"
	//         "LineNumber"
	//         "Message"
}

// TestData ...
type TestData struct {
	XMLName    xml.Name `xml:"testsuite"`
	Name       string   `xml:"name,attr"`
	Tests      int      `xml:"tests,attr"`
	Failures   int      `xml:"failures,attr"`
	Errors     int      `xml:"errors,attr"`
	Skipped    int      `xml:"skipped,attr"`
	Time       float32  `xml:"time,attr"`
	Timestamp  string   `xml:"timestamp,attr"`
	Hostname   string   `xml:"hostname"`
	Properties []struct {
		Name  string `xml:"name,attr"`
		Value string `xml:"value,attr"`
	} `xml:"properties>property"`
	TestCases []TestCase `xml:"testcase"`
}

type TestCase struct {
	Name      string  `xml:"name,attr"`
	ClassName string  `xml:"classname,attr"`
	Time      float32 `xml:"time,attr"`
	Failure   *string `xml:"failure,omitempty"`
	Skipped   *string `xml:"skipped,omitempty"`
	Error     *string `xml:"error,omitempty"`
	// Don't forget to work with Errors and Skipped
}

type TestableSummary struct {
	TestName string `plist:"TestName"`
	Tests    []Test `plist:"Tests"`
}

type Test struct {
	SubTests []SubTest `plist:"Subtests"`
}

type SubTest struct {
	TestID           string           `plist:"TestIdentifier"`
	TestStatus       string           `plist:"TestStatus"`
	Duration         float32          `plist:"Duration"`
	FailureSummaries []FailureSummary `plist:"FailureSummaries"`
	SubTests         []SubTest        `plist:"Subtests"`
}

type FailureSummary struct {
	FileName   string `plist:"FileName"`
	LineNumber int    `plist:"LineNumber"`
	Message    string `plist:"Message"`
}

type XCTests struct {
	TestableSummaries []TestableSummary `plist:"TestableSummaries"`
}

func (xcTests XCTests) CleanSubTests() XCTests {
	xc := XCTests{TestableSummaries: xcTests.TestableSummaries}

	for si, summary := range xc.TestableSummaries {
		for ti, test := range summary.Tests {
			xc.TestableSummaries[si].Tests[ti].SubTests = cleanRecursiveSubTest(test.SubTests)
		}
	}

	return xc
}

func cleanRecursiveSubTest(subTests []SubTest) []SubTest {
	if subTests[0].SubTests != nil {
		return cleanRecursiveSubTest(subTests[0].SubTests)
	}
	return subTests
}
