package autodetect

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type mavenInjecter struct{}

func newMavenInjecter() Injecter {
	return &mavenInjecter{}
}

const (
	mavenBuildCacheConfig = `<cache>
   <configuration>
       <validateXml>true</validateXml>
       <remote enabled="true" saveToRemote="true" id="cache-remote-server">
           <url>%s/cache/maven</url>
       </remote>
   </configuration>
</cache>`

	mavenExtensionsXml = `<extensions>
    <extension>
        <groupId>org.apache.maven.extensions</groupId>
        <artifactId>maven-build-cache-extension</artifactId>
        <version>1.2.0</version>
    </extension>
</extensions>`
)

func writeXMLIfNotExists(filepath string, content string) error {
	// Check if file exists
	if _, err := os.Stat(filepath); err == nil {
		// File exists, skip
		fmt.Printf("File %s already exists, ..skipping configuring Harness Build Intelligence.\n", filepath)
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("error checking file %s: %w", filepath, err)
	}

	// Create a temporary buffer to format the XML
	var buf bytes.Buffer
	dec := xml.NewDecoder(strings.NewReader(content))
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")

	// Copy and format the XML
	for {
		token, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error parsing XML content: %w", err)
		}
		err = enc.EncodeToken(token)
		if err != nil {
			return fmt.Errorf("error formatting XML content: %w", err)
		}
	}

	// Flush the encoder
	if err := enc.Flush(); err != nil {
		return fmt.Errorf("error flushing XML encoder: %w", err)
	}

	// Write the file
	if err := os.WriteFile(filepath, []byte(xml.Header+buf.String()), 0644); err != nil {
		return fmt.Errorf("error writing file %s: %w", filepath, err)
	}

	fmt.Printf("Created new file %s\n", filepath)
	return nil
}

func (m *mavenInjecter) InjectTool() error {
	// Create .mvn directory
	mvnDir := ".mvn"
	if err := os.MkdirAll(mvnDir, 0755); err != nil {
		return fmt.Errorf("failed to create .mvn directory: %w", err)
	}

	// Try to create maven-build-cache-config.xml
	cacheConfigPath := filepath.Join(mvnDir, "maven-build-cache-config.xml")
	cacheProxyEndpoint := os.Getenv("HARNESS_CACHE_PROXY_ENDPOINT")
	mavenBuildCacheConfigContent := fmt.Sprintf(mavenBuildCacheConfig, cacheProxyEndpoint)
	if err := writeXMLIfNotExists(cacheConfigPath, mavenBuildCacheConfigContent); err != nil {
		return fmt.Errorf("failed to create maven-build-cache-config.xml: %w", err)
	}

	// Try to create extensions.xml
	extensionsPath := filepath.Join(mvnDir, "extensions.xml")
	if err := writeXMLIfNotExists(extensionsPath, mavenExtensionsXml); err != nil {
		return fmt.Errorf("failed to create extensions.xml: %w", err)
	}

	fmt.Println("Successfully injected Maven build cache configuration")
	return nil
}
