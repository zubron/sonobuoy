// +build integration

package integration

import (
	"bytes"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
var sonobuoy string
var gitSHA string
var gitVersion string

func stringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func randomNamespace() string {
	charset := "abcdefghijklmnopqrstuvwxyz"
	return "integration-" + stringWithCharset(5, charset)
}

func findSonobuoyCLI() (string, error) {
	sonobuoyPath := os.Getenv("SONOBUOY_CLI")
	if sonobuoyPath == "" {
		sonobuoyPath = "../../sonobuoy"
	}
	if _, err := os.Stat(sonobuoyPath); os.IsNotExist(err) {
		return "", err
	}

	return sonobuoyPath, nil
}

func TestSonobuoyVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer

	command := exec.Command(sonobuoy, "version2", "--kubeconfig", os.Getenv("KUBECONFIG"))
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				t.Logf("exit code: %d", status.ExitStatus())
			}
		}
		t.Errorf("Unexpected error running command: %v", err)
		t.Log(stderr.String())
		t.FailNow()
	}

	expectedOutput := []string{
		fmt.Sprintf("Sonobuoy Version: %v", gitVersion),
		"MinimumKubeVersion: 1.13.0",
		"MaximumKubeVersion: 1.15.99",
		fmt.Sprintf("GitSHA: %v", gitSHA),
	}

	lines := strings.Split(stdout.String(), "\n")
	for i, line := range lines {
		if i < len(expectedOutput) {
			if expectedOutput[i] != line {
				t.Errorf("unexpected output in version information, expected %v, got %v", expectedOutput[i], line)
			}
		}
	}
}

// func TestSonobuoyRunQuick(t *testing.T) {
// 	g := gomega.NewGomegaWithT(t)
// 	namespace := randomNamespace()
//
// 	var stdout bytes.Buffer
// 	var stderr bytes.Buffer
// 	command := exec.Command(sonobuoy, "run", "--mode=quick", "--wait", "-n", namespace)
// 	session, err := gexec.Start(command, &stdout, &stderr)
// 	g.Expect(err).ShouldNot(gomega.HaveOccurred())
// 	g.Eventually(session, 600).Should(gexec.Exit(1))
// 	t.Log(stdout.String())
// 	t.Log(stderr.String())
//
// 	cleanupCommand := exec.Command("../../sonobuoy", "delete", "--wait", "-n", namespace)
// 	csession, cerr := gexec.Start(cleanupCommand, &stdout, &stderr)
// 	g.Expect(cerr).ShouldNot(gomega.HaveOccurred())
// 	g.Eventually(csession, 600).Should(gexec.Exit(0))
// 	t.Log(stdout.String())
// 	t.Log(stderr.String())
// }

func TestMain(m *testing.M) {
	var err error
	sonobuoy, err = findSonobuoyCLI()
	if err != nil {
		fmt.Printf("Skipping integration tests: failed to find sonobuoy CLI: %v\n", err)
		os.Exit(1)
	}

	flag.StringVar(&gitSHA, "git-sha", os.Getenv("GIT_SHA"), "SHA of HEAD Git commit (git rev-parse --verify HEAD)")
	flag.StringVar(&gitVersion, "git-version", os.Getenv("GIT_VERSION"), "Git version used for Sonobuoy (git describe --always --dirty --tags)")
	flag.Parse()

	if gitSHA == "" {
		fmt.Println("Git SHA must be provided, using --git-sha or setting environment variable GIT_SHA")
		os.Exit(1)
	}
	if gitVersion == "" {
		fmt.Println("Sonobuoy version from git must be provided, using --git-version or setting environment variable GIT_VERSION")
		os.Exit(1)
	}

	result := m.Run()
	os.Exit(result)
}
