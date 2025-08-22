package autodetect

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type bazelInjecter struct{}

func newBazelInjecter() *bazelInjecter {
	return &bazelInjecter{}
}

func (*bazelInjecter) InjectTool() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return errors.New("error getting user home directory")
	}

	bazelrcFile := filepath.Join(homeDir, ".bazelrc")
	cacheProxyEndpoint := os.Getenv("HARNESS_CACHE_PROXY_ENDPOINT")

	bazelrcContent := fmt.Sprintf(`build --remote_cache=%s/cache/bazel`, cacheProxyEndpoint)

	err = WriteOrAppendToFile(bazelrcFile, bazelrcContent)
	if err != nil {
		return fmt.Errorf("error writing to bazelrc file: %w", err)
	}
	return nil
}
