package autodetect

import (
	"crypto/md5" // #nosec
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type buildToolInfo struct {
	globToDetect string
	tool         string
	injecter     Injecter
}

func DetectDirectoriesToCache(path string) error {
	fmt.Printf("auto-detect: scanning path=%s\n", path)
	var buildToolInfoMapping = []buildToolInfo{
		{
			globToDetect: "build.gradle",
			tool:         "gradle",
			injecter:     newGradleInjecter(),
		},
		{
			globToDetect: "build.gradle.kts",
			tool:         "gradle",
			injecter:     newGradleInjecter(),
		},
		{
			globToDetect: "WORKSPACE",
			tool:         "bazel",
			injecter:     newBazelInjecter(),
		},
		{
			globToDetect: "pom.xml",
			tool:         "maven",
			injecter:     newMavenInjecter(),
		},
	}

	for _, supportedTool := range buildToolInfoMapping {
		fmt.Printf("auto-detect: checking tool=%s marker=%s\n", supportedTool.tool, supportedTool.globToDetect)
		hash, dir, err := hashIfFileExist(path, supportedTool.globToDetect)
		if err != nil {
			fmt.Printf("auto-detect: error during hashIfFileExist(root) for marker=%s: %v\n", supportedTool.globToDetect, err)
			return err
		}
		if hash == "" {
			fmt.Printf("auto-detect: no root-level match for %s, attempting recursive search\n", supportedTool.globToDetect)
			hash, dir, err = hashIfFileExist(path, filepath.Join("**", supportedTool.globToDetect))
		}
		if err != nil {
			fmt.Printf("auto-detect: error during hashIfFileExist(recursive) for marker=%s: %v\n", supportedTool.globToDetect, err)
			return err
		}
		if dir != "" && hash != "" {
			fmt.Printf("auto-detect: detected tool=%s at dir=%s hash=%s\n", supportedTool.tool, dir, hash)
			// Ensure workspace-level cache directory for Gradle (e.g., /harness/.gradle)
			if supportedTool.tool == "gradle" {
				workspaceGradle := filepath.Join(path, ".gradle")
				fmt.Printf("auto-detect: ensuring workspace gradle dir: %s\n", workspaceGradle)
				if mkErr := os.MkdirAll(workspaceGradle, 0755); mkErr != nil {
					return fmt.Errorf("failed to create workspace gradle dir %s: %w", workspaceGradle, mkErr)
				}
				fmt.Printf("auto-detect: ensured workspace gradle dir exists: %s\n", workspaceGradle)
			}
			fmt.Printf("auto-detect: invoking injector for tool=%s\n", supportedTool.tool)
			err = supportedTool.injecter.InjectTool()
			if err != nil {
				fmt.Printf("Error while auto-injecting for %s build tool: %s\n", supportedTool.tool, err.Error())
				continue
			}
			fmt.Printf("auto-detect: injector completed successfully for tool=%s\n", supportedTool.tool)
		} else {
			fmt.Printf("auto-detect: no marker found for tool=%s\n", supportedTool.tool)
		}
	}
	return nil
}

func hashIfFileExist(path, glob string) (string, string, error) {
	pattern := filepath.Join(path, glob)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		fmt.Printf("auto-detect: invalid glob pattern pattern=%s error=%v\n", pattern, err)
		return "", "", err
	}
	fmt.Printf("auto-detect: glob pattern=%s matches=%d\n", pattern, len(matches))
	if len(matches) == 0 {
		return "", "", nil
	}

	return calculateMd5FromFiles(matches)
}

func calculateMd5FromFiles(fileList []string) (string, string, error) {
	fmt.Printf("auto-detect: calculating md5 from files: %v\n", fileList)
	rootMostFile := shortestPath(fileList)
	fmt.Printf("auto-detect: chosen root-most file: %s\n", rootMostFile)
	file, err := os.Open(rootMostFile)

	if err != nil {
		fmt.Printf("auto-detect: error opening file %s: %v\n", rootMostFile, err)
		return "", "", err
	}

	dir, err := filepath.Abs(filepath.Dir(rootMostFile))

	if err != nil {
		fmt.Printf("auto-detect: error resolving abs dir for %s: %v\n", rootMostFile, err)
		return "", "", err
	}

	defer file.Close()

	if err != nil {
		return "", "", err
	}

	hash := md5.New() // #nosec
	_, err = io.Copy(hash, file)

	if err != nil {
		fmt.Printf("auto-detect: error hashing file %s: %v\n", rootMostFile, err)
		return "", "", err
	}

	sum := hex.EncodeToString(hash.Sum(nil))
	fmt.Printf("auto-detect: md5 calculated for %s: %s (dir=%s)\n", rootMostFile, sum, dir)
	return sum, dir, nil
}

func shortestPath(input []string) (shortest string) {
	size := len(input[0])
	for _, v := range input {
		if len(v) <= size {
			shortest = v
			size = len(v)
		}
	}

	return
}
