package main

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
)

func filterJUnitTestResults(files *[]string) ([]string, error) {
	filteredFiles := []string{}
	for _, file := range *files {
		f, err := os.Open(file)
		if err != nil {
			return nil, err
		}

		scanner := bufio.NewReader(f)

		for i := 0; i < 10; i++ {
			line, _, _ := scanner.ReadLine()
			if bytes.Contains(line, []byte("<testsuite")) {
				filteredFiles = append(filteredFiles, file)
				break
			}
		}
	}
	return filteredFiles, nil
}

func filterXcodeTestResults(files *[]string) ([]string, error) {
	filteredFiles := []string{}
	for _, file := range *files {
		if strings.HasSuffix(strings.ToLower(file), "testsummaries.plist") {
			filteredFiles = append(filteredFiles, file)
			break
		}
	}
	return filteredFiles, nil
}

func getFilesByExt(ext string, files *[]string) func(osFilePath string, info os.FileInfo, err error) error {
	return func(osPathname string, info os.FileInfo, err error) error {
		if filepath.Ext(info.Name()) == "."+ext {
			*files = append(*files, osPathname)
		}
		return err
	}
}
