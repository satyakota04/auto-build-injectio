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

	// Debug environment visibility (avoid printing secrets)
	fmt.Printf("gradle: env HARNESS_ACCOUNT_ID set=%t HARNESS_CACHE_SERVICE_TOKEN set=%t HARNESS_CACHE_PROXY_ENDPOINT set=%t\n", accountID != "", bearerToken != "", endpoint != "")
	fmt.Printf("gradle: HARNESS_GRADLE_PLUGIN_VERSION=%q HARNESS_CACHE_PUSH=%q HARNESS_CACHE_LOCAL_ENABLED=%q\n", gradlePluginVersion, gradleCachePush, localCacheEnabled)

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
		fmt.Printf("gradle: injecting into GRADLE_HOME=%s\n", gradleHome)
		if e := injectGradleFiles(gradleHome, initGradleContent, gradlePropertiesContent); e != nil {
			fmt.Printf("gradle: error injecting into GRADLE_HOME=%s: %v\n", gradleHome, e)
		} else {
			fmt.Printf("gradle: injection successful for GRADLE_HOME=%s\n", gradleHome)
		}
	} else {
		fmt.Printf("gradle: GRADLE_HOME not set; skipping\n")
	}

	// For $GRADLE_USER_HOME
	gradleUserHome := os.Getenv("GRADLE_USER_HOME")
	if gradleUserHome != "" {
		fmt.Printf("gradle: injecting into GRADLE_USER_HOME=%s\n", gradleUserHome)
		if e := injectGradleFiles(gradleUserHome, initGradleContent, gradlePropertiesContent); e != nil {
			fmt.Printf("gradle: error injecting into GRADLE_USER_HOME=%s: %v\n", gradleUserHome, e)
		} else {
			fmt.Printf("gradle: injection successful for GRADLE_USER_HOME=%s\n", gradleUserHome)
		}
	} else {
		fmt.Printf("gradle: GRADLE_USER_HOME not set; skipping\n")
	}

	// For ~/.gradle
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("error getting user home directory: %w", err)
	}
	fmt.Printf("gradle: injecting into $HOME/.gradle with HOME=%s\n", homeDir)
	gradleDir := filepath.Join(homeDir, ".gradle")
	if e := injectGradleFiles(gradleDir, initGradleContent, gradlePropertiesContent); e != nil {
		fmt.Printf("gradle: error injecting into HOME/.gradle=%s: %v\n", gradleDir, e)
	} else {
		fmt.Printf("gradle: injection successful for HOME/.gradle=%s\n", gradleDir)
	}

	// for sudo command
	sudoDir := "/root"
	sudoGradleDir := filepath.Join(sudoDir, ".gradle")
	fmt.Printf("gradle: best-effort injection into %s (may fail without permissions)\n", sudoGradleDir)
	if e := injectGradleFiles(sudoGradleDir, initGradleContent, gradlePropertiesContent); e != nil {
		fmt.Printf("gradle: best-effort /root injection skipped: %v\n", e)
	} else {
		fmt.Printf("gradle: best-effort /root injection successful: %s\n", sudoGradleDir)
	}

	return nil
}

func injectGradleFiles(gradleDir string, initGradleContent string, gradlePropertiesContent string) error {
	fmt.Printf("gradle: injectGradleFiles start dir=%s\n", gradleDir)
	err := os.MkdirAll(gradleDir, os.ModePerm)
	if err != nil {
		fmt.Printf("gradle: error creating %s directory: %v\n", gradleDir, err)
		return fmt.Errorf("error creating %s directory: %w", gradleDir, err)
	}
	fmt.Printf("gradle: ensured dir exists: %s\n", gradleDir)
	// $gradleDir/init.gradle file
	gradleHomeInit := filepath.Join(gradleDir, "init.d")
	initGradleHomeFile := filepath.Join(gradleHomeInit, "init.gradle")
	fmt.Printf("gradle: writing %s\n", initGradleHomeFile)
	err = WriteOrAppendToFile(initGradleHomeFile, initGradleContent)
	if err != nil {
		fmt.Printf("gradle: error writing to %s: %v\n", initGradleHomeFile, err)
		return fmt.Errorf("error writing to %s file: %w", initGradleHomeFile, err)
	}
	// gradleDir/gradle.properties file
	gradleHomePropertiesFile := filepath.Join(gradleDir, "gradle.properties")
	fmt.Printf("gradle: writing %s\n", gradleHomePropertiesFile)
	err = WriteOrAppendToFile(gradleHomePropertiesFile, gradlePropertiesContent)
	if err != nil {
		fmt.Printf("gradle: error writing to %s: %v\n", gradleHomePropertiesFile, err)
		return fmt.Errorf("error writing to %s file: %w", gradleHomePropertiesFile, err)
	}

	fmt.Printf("gradle: injectGradleFiles done dir=%s\n", gradleDir)
	return nil
}
