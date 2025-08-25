package autodetect

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type gradleInjecter struct{}

func newGradleInjecter() *gradleInjecter {
	return &gradleInjecter{}
}

func (*gradleInjecter) InjectTool() error {
	accountID := os.Getenv("HARNESS_ACCOUNT_ID")
	bearerToken := os.Getenv("HARNESS_CACHE_SERVICE_TOKEN")
	endpoint := os.Getenv("HARNESS_CACHE_PROXY_ENDPOINT")
	gradlePluginVersion := os.Getenv("HARNESS_GRADLE_PLUGIN_VERSION")
	gradleCachePush := os.Getenv("HARNESS_CACHE_PUSH")
	localCacheEnabled := os.Getenv("HARNESS_CACHE_LOCAL_ENABLED")

	// Check if environment variables are set
	if accountID == "" || bearerToken == "" || endpoint == "" {
		return errors.New("please set HARNESS_ACCOUNT_ID,HARNESS_CACHE_SERVICE_TOKEN, and HARNESS_CACHE_PROXY_ENDPOINT")
	}

	// Define the content to be written to gradle.properties
	gradlePropertiesContent := "org.gradle.caching=true\n"

	// Define the content to be written to init.gradle
	initGradleContent := fmt.Sprintf(`
initscript {
    repositories {
		if (System.getenv('MAVEN_URL')) {
            maven {
                url System.getenv('MAVEN_URL')
            }
        } else {
			mavenCentral()
		}       
    }
    dependencies {
        classpath 'io.harness:gradle-cache:%s'
    }
}
// Apply the plugin to the Settings object
gradle.settingsEvaluated { settings ->
    settings.pluginManager.apply(io.harness.HarnessBuildCache)
    settings.buildCache {
            local {
                enabled = "%s"
            }
            remote(io.harness.Cache) {
                accountId = System.getenv('HARNESS_ACCOUNT_ID')
                push = "%s"
                endpoint = System.getenv('HARNESS_CACHE_PROXY_ENDPOINT')
            }
        }
}
`, gradlePluginVersion, localCacheEnabled, gradleCachePush)

	// Injecting the above configs in gradle files
	// For $GRADLE_HOME
	gradleHome := os.Getenv("GRADLE_HOME")
	if gradleHome != "" {
		// $GRADLE_HOME/init.d/init.gradle file
		injectGradleFiles(gradleHome, initGradleContent, gradlePropertiesContent)
	}

	// For $GRADLE_USER_HOME
	gradleUserHome := os.Getenv("GRADLE_USER_HOME")
	if gradleUserHome != "" {
		// $GRADLE_USER_HOME/init.d/init.gradle file
		injectGradleFiles(gradleUserHome, initGradleContent, gradlePropertiesContent)
	}

	// For ~/.gradle
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("error getting user home directory: %w", err)
	}

	gradleDir := filepath.Join(homeDir, ".gradle")
	injectGradleFiles(gradleDir, initGradleContent, gradlePropertiesContent)

	// for sudo command
	sudoDir := "/root"
	sudoGradleDir := filepath.Join(sudoDir, ".gradle")
	injectGradleFiles(sudoGradleDir, initGradleContent, gradlePropertiesContent)

	return nil
}

func injectGradleFiles(gradleDir string, initGradleContent string, gradlePropertiesContent string) error {
	err := os.MkdirAll(gradleDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error creating %s directory: %w", gradleDir, err)
	}
	// $gradleDir/init.gradle file
	gradleHomeInit := filepath.Join(gradleDir, "init.d")
	initGradleHomeFile := filepath.Join(gradleHomeInit, "init.gradle")
	err = WriteOrAppendToFile(initGradleHomeFile, initGradleContent)
	if err != nil {
		return fmt.Errorf("error writing to %s file: %w", initGradleContent, err)
	}
	// gradleDir/gradle.properties file
	gradleHomePropertiesFile := filepath.Join(gradleDir, "gradle.properties")
	err = WriteOrAppendToFile(gradleHomePropertiesFile, gradlePropertiesContent)
	if err != nil {
		return fmt.Errorf("error writing to %s file: %w", gradlePropertiesContent, err)
	}

	return nil
}
