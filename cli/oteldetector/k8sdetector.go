package oteldetector

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/semconv"
)

const (
	k8sTokenPath      = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	k8sCertPath       = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	defaultCgroupPath = "/proc/self/cgroup"
	containerIDLength = 64
	timeoutMillis     = 2000
)

type k8sDetector struct{}

func NewK8sDetector() resource.Detector {
	return &k8sDetector{}
}

func (k8sDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	if !isK8s() {
		return nil, nil
	}
	podinfo := "/etc/podinfo"
	info, err := os.Stat(podinfo)
	if err != nil || !info.IsDir() {
		fmt.Println("cannot find podinfo")
		return nil, nil
	}

	var attrs []attribute.KeyValue
	namespace := readFile(filepath.Join(podinfo, "namespace"))
	if namespace != "" {
		attrs = append(attrs, semconv.K8SNamespaceNameKey.String(namespace))
	}

	podname := readFile(filepath.Join(podinfo, "name"))
	if podname != "" {
		attrs = append(attrs, semconv.K8SPodNameKey.String(podname))
	}

	poduid := readFile(filepath.Join(podinfo, "uid"))
	if poduid != "" {
		attrs = append(attrs, semconv.K8SPodUIDKey.String(poduid))
	}

	return resource.NewWithAttributes(attrs...), nil
}

func readFile(path string) string {
	d, err := ioutil.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(d)
}

// isK8s checks if the current environment is running in a Kubernetes environment
func isK8s() bool {
	return fileExists(k8sTokenPath) && fileExists(k8sCertPath)
}

// getContainerID returns the containerID if currently running within a container.
func getContainerID() (string, error) {
	fileData, err := ioutil.ReadFile(defaultCgroupPath)
	if err != nil {
		return "", fmt.Errorf("getContainerID() error: cannot read file with path %s: %w", defaultCgroupPath, err)
	}

	r, err := regexp.Compile(`^.*/docker/(.+)$`)
	if err != nil {
		return "", err
	}

	fmt.Println("fileData", string(fileData))
	// Retrieve containerID from file
	splitData := strings.Split(strings.TrimSpace(string(fileData)), "\n")
	for _, str := range splitData {
		if r.MatchString(str) {
			return str[len(str)-containerIDLength:], nil
		}
	}
	return "", fmt.Errorf("getContainerID() error: cannot read containerID from file %s", defaultCgroupPath)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
