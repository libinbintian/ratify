package plugin

import (
	"context"
	"strings"
	"testing"

	"github.com/deislabs/ratify/pkg/common"
	"github.com/deislabs/ratify/pkg/ocispecs"
	"github.com/deislabs/ratify/pkg/verifier/config"
	"github.com/deislabs/ratify/pkg/verifier/mocks"
)

type TestExecutor struct {
	find    func(plugin string, paths []string) (string, error)
	execute func(ctx context.Context, pluginPath string, cmdArgs []string, stdinData []byte, environ []string) ([]byte, error)
}

func (e *TestExecutor) ExecutePlugin(ctx context.Context, pluginPath string, cmdArgs []string, stdinData []byte, environ []string) ([]byte, error) {
	return e.execute(ctx, pluginPath, cmdArgs, stdinData, environ)
}

func (e *TestExecutor) FindInPaths(plugin string, paths []string) (string, error) {
	return e.find(plugin, paths)
}

func TestNewVerifier_Expected(t *testing.T) {
	var verifierConfig config.VerifierConfig
	verifierConfig = map[string]interface{}{
		"name":             "test-verifier",
		"artifactTypes":    "test1,test2",
		"nestedReferences": "ref1,ref2",
	}

	verifier, err := NewVerifier("1.0.0", verifierConfig, []string{})
	if err != nil {
		t.Fatalf("failed to create plugin store %v", err)
	}

	if vc, ok := verifier.(*VerifierPlugin); !ok {
		t.Fatal("type assertion failed. expected plugin verifier")
	} else if len(vc.artifactTypes) != 2 {
		t.Fatalf("expected number of artifact Types 2, actual %d", len(vc.artifactTypes))
	} else if len(vc.nestedReferences) != 2 {
		t.Fatalf("expected number of nested references is 2, actual %d", len(vc.nestedReferences))
	}
}

func TestVerify_NoNestedReferences_Expected(t *testing.T) {
	testPlugin := "test-plugin"
	testExecutor := &TestExecutor{
		find: func(plugin string, paths []string) (string, error) {
			return "testpath", nil
		},
		execute: func(ctx context.Context, pluginPath string, cmdArgs []string, stdinData []byte, environ []string) ([]byte, error) {
			if pluginPath != "testpath" {
				t.Fatalf("mismatch in plugin path expected %s actual %s", "testpath", pluginPath)
			}
			if cmdArgs != nil {
				t.Fatal("cmdArgs is expected to be nil")
			}
			stdData := string(stdinData[:])
			if !strings.Contains(stdData, testPlugin) || !strings.Contains(stdData, "test-type") {
				t.Fatalf("missing config data in stdin expected to have %s actual %s", "test-plugin, test-type", stdData)
			}

			commandCheck := false
			versionCheck := false
			subjectCheck := false
			for _, env := range environ {
				if strings.Contains(env, CommandEnvKey) && strings.Contains(env, VerifyCommand) {
					commandCheck = true
				} else if strings.Contains(env, VersionEnvKey) && strings.Contains(env, "1.0.0") {
					versionCheck = true
				} else if strings.Contains(env, SubjectEnvKey) && strings.Contains(env, "localhost") {
					subjectCheck = true
				}

			}

			if !commandCheck {
				t.Fatalf("missing command env")
			}

			if !versionCheck {
				t.Fatalf("missing version env")
			}

			if !subjectCheck {
				t.Fatalf("missing subject env")
			}

			verifierResult := ` {"isSuccess":true}`
			return []byte(verifierResult), nil
		},
	}

	var verifierConfig config.VerifierConfig
	verifierConfig = map[string]interface{}{
		"name": testPlugin,
	}
	verifierPlugin := &VerifierPlugin{
		name:          testPlugin,
		artifactTypes: []string{"test-type"},
		version:       "1.0.0",
		executor:      testExecutor,
		rawConfig:     verifierConfig,
	}

	subject := common.Reference{
		Original: "localhost",
	}
	ref := ocispecs.ReferenceDescriptor{
		ArtifactType: "test-type",
	}

	result, err := verifierPlugin.Verify(context.Background(), subject, ref, &mocks.TestStore{}, &mocks.TestExecutor{})

	if err != nil {
		t.Fatalf("plugin execution failed %v", err)
	}

	if !result.IsSuccess {
		t.Fatal("plugin expected to return isSuccess as true but got as false")
	}
}

func TestVerify_NestedReferences_Verify_Failed(t *testing.T) {
	testPlugin := "test-plugin"

	var verifierConfig config.VerifierConfig
	verifierConfig = map[string]interface{}{
		"name": testPlugin,
	}
	verifierPlugin := &VerifierPlugin{
		name:             testPlugin,
		artifactTypes:    []string{"test-type"},
		nestedReferences: []string{"type1"},
		version:          "1.0.0",
		executor:         &TestExecutor{},
		rawConfig:        verifierConfig,
	}

	subject := common.Reference{
		Original: "localhost",
	}
	ref := ocispecs.ReferenceDescriptor{
		ArtifactType: "test-type",
	}

	result, err := verifierPlugin.Verify(context.Background(), subject, ref, &mocks.TestStore{}, &mocks.TestExecutor{VerifySuccess: false})

	if err != nil {
		t.Fatalf("plugin execution failed %v", err)
	}

	if result.IsSuccess {
		t.Fatal("plugin expected to return isSuccess as false but got as true")
	}

	if len(result.NestedResults) != 1 {
		t.Fatalf("plugin expected to return single nested result but returned count is %d", len(result.NestedResults))
	}
}

func TestVerify_NestedReferences_Verify_Success(t *testing.T) {
	testPlugin := "test-plugin"
	testExecutor := &TestExecutor{
		find: func(plugin string, paths []string) (string, error) {
			return "testpath", nil
		},
		execute: func(ctx context.Context, pluginPath string, cmdArgs []string, stdinData []byte, environ []string) ([]byte, error) {
			if pluginPath != "testpath" {
				t.Fatalf("mismatch in plugin path expected %s actual %s", "testpath", pluginPath)
			}
			if cmdArgs != nil {
				t.Fatal("cmdArgs is expected to be nil")
			}
			stdData := string(stdinData[:])
			if !strings.Contains(stdData, testPlugin) || !strings.Contains(stdData, "test-type") {
				t.Fatalf("missing config data in stdin expected to have %s actual %s", "test-plugin, test-type", stdData)
			}

			commandCheck := false
			versionCheck := false
			subjectCheck := false
			for _, env := range environ {
				if strings.Contains(env, CommandEnvKey) && strings.Contains(env, VerifyCommand) {
					commandCheck = true
				} else if strings.Contains(env, VersionEnvKey) && strings.Contains(env, "1.0.0") {
					versionCheck = true
				} else if strings.Contains(env, SubjectEnvKey) && strings.Contains(env, "localhost") {
					subjectCheck = true
				}

			}

			if !commandCheck {
				t.Fatalf("missing command env")
			}

			if !versionCheck {
				t.Fatalf("missing version env")
			}

			if !subjectCheck {
				t.Fatalf("missing subject env")
			}

			verifierResult := ` {"isSuccess":true}`
			return []byte(verifierResult), nil
		},
	}
	var verifierConfig config.VerifierConfig
	verifierConfig = map[string]interface{}{
		"name": testPlugin,
	}
	verifierPlugin := &VerifierPlugin{
		name:             testPlugin,
		artifactTypes:    []string{"test-type"},
		nestedReferences: []string{"type1"},
		version:          "1.0.0",
		executor:         testExecutor,
		rawConfig:        verifierConfig,
	}

	subject := common.Reference{
		Original: "localhost",
	}
	ref := ocispecs.ReferenceDescriptor{
		ArtifactType: "test-type",
	}

	result, err := verifierPlugin.Verify(context.Background(), subject, ref, &mocks.TestStore{}, &mocks.TestExecutor{VerifySuccess: true})

	if err != nil {
		t.Fatalf("plugin execution failed %v", err)
	}

	if !result.IsSuccess {
		t.Fatal("plugin expected to return isSuccess as true but got as false")
	}

	if len(result.NestedResults) != 1 {
		t.Fatalf("plugin expected to return single nested result but returned count is %d", len(result.NestedResults))
	}

	if !result.NestedResults[0].IsSuccess {
		t.Fatal("plugin expected to return isSuccess as true for nested result but got as false")
	}
}