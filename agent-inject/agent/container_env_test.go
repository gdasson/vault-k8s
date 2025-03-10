// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package agent

import (
	"testing"

	"github.com/hashicorp/vault/sdk/helper/strutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestContainerEnvs(t *testing.T) {

	tests := []struct {
		agent        Agent
		expectedEnvs []string
	}{
		{Agent{}, []string{"VAULT_CONFIG"}},
		{Agent{Vault: Vault{Address: "http://localhost:8200"}, ConfigMapName: "foobar"}, []string{"VAULT_SKIP_VERIFY", "VAULT_ADDR"}},
		{Agent{Vault: Vault{ClientMaxRetries: "0"}}, []string{"VAULT_CONFIG", "VAULT_MAX_RETRIES"}},
		{Agent{Vault: Vault{ClientTimeout: "5s"}}, []string{"VAULT_CONFIG", "VAULT_CLIENT_TIMEOUT"}},
		{Agent{Vault: Vault{ClientMaxRetries: "0", ClientTimeout: "5s"}}, []string{"VAULT_CONFIG", "VAULT_MAX_RETRIES", "VAULT_CLIENT_TIMEOUT"}},
		{Agent{ConfigMapName: "foobar", Vault: Vault{Address: "http://localhost:8200", ClientMaxRetries: "0", ClientTimeout: "5s", LogLevel: "info", ProxyAddress: "http://proxy:3128"}}, []string{"VAULT_MAX_RETRIES", "VAULT_CLIENT_TIMEOUT", "VAULT_LOG_LEVEL", "HTTPS_PROXY", "VAULT_SKIP_VERIFY", "VAULT_ADDR"}},
		{Agent{Vault: Vault{GoMaxProcs: "1"}}, []string{"VAULT_CONFIG", "GOMAXPROCS"}},
		{Agent{Vault: Vault{CACert: ""}, VaultCACertInjectorSupplied: "DummyPEMCertificate"}, []string{"VAULT_CONFIG", "CACERT"}},
		{Agent{Vault: Vault{CACert: "/home/vault/cacert.pem"}, VaultCACertInjectorSupplied: "DummyPEMCertificate"}, []string{"VAULT_CONFIG"}},
		{Agent{Vault: Vault{CACert: "/home/vault/cacert.pem"}, VaultCACertInjectorSupplied: ""}, []string{"VAULT_CONFIG"}},
		{Agent{Vault: Vault{CACert: "/home/vault/cacert.pem"}, ConfigMapName: "foobar", VaultCACertInjectorSupplied: ""}, []string{"VAULT_CACERT", "VAULT_SKIP_VERIFY"}},
	}

	for _, tt := range tests {
		envs, err := tt.agent.ContainerEnvVars(true)
		if err != nil {
			t.Errorf("got error, shouldn't have: %s", err)
		}
		if len(envs) != len(tt.expectedEnvs) {
			t.Errorf("number of envs mismatch, wanted %d, got %d", len(tt.expectedEnvs), len(envs))
		}

		for _, env := range envs {
			if !strutil.StrListContains(tt.expectedEnvs, env.Name) {
				t.Errorf("unexpected env found %s", env.Name)
			}
		}
	}
}

func TestContainerEnvsForIRSA(t *testing.T) {
	envTests := []struct {
		agent        Agent
		expectedEnvs []string
	}{
		{Agent{Pod: testPodWithoutIRSA()}, []string{"VAULT_CONFIG"}},
		{Agent{Pod: testPodWithIRSA(), Vault: Vault{AuthType: "aws"}},
			[]string{"VAULT_CONFIG", "AWS_ROLE_ARN", "AWS_WEB_IDENTITY_TOKEN_FILE", "AWS_DEFAULT_REGION", "AWS_REGION"},
		},
	}
	for _, tt := range envTests {
		envs, err := tt.agent.ContainerEnvVars(true)
		if err != nil {
			t.Errorf("got error, shouldn't have: %s", err)
		}
		if len(envs) != len(tt.expectedEnvs) {
			t.Errorf("number of envs mismatch, wanted %d, got %d", len(tt.expectedEnvs), len(envs))
		}
	}
}

func TestAwsRegionEnvForAwsAuthMethod(t *testing.T) {
	input := []struct {
		agent        Agent
		expectedEnvs []string
	}{
		{Agent{Pod: testPodWithRegionInAuthConfig(), Vault: Vault{AuthType: "aws", AuthConfig: getRegionMap()}},
			[]string{"VAULT_CONFIG", "AWS_ROLE_ARN", "AWS_WEB_IDENTITY_TOKEN_FILE", "AWS_REGION"},
		},
		{Agent{Pod: testPodWithIRSA(), Vault: Vault{AuthType: "aws"}},
			[]string{"VAULT_CONFIG", "AWS_ROLE_ARN", "AWS_WEB_IDENTITY_TOKEN_FILE", "AWS_DEFAULT_REGION", "AWS_REGION"},
		},
	}
	for _, item := range input {
		envs, err := item.agent.ContainerEnvVars(true)
		if err != nil {
			t.Errorf("got error, shouldn't have: %s", err)
		}
		if len(envs) != len(item.expectedEnvs) {
			t.Errorf("number of envs mismatch, wanted %d, got %d", len(item.expectedEnvs), len(envs))
		}
	}
}

func getRegionMap() map[string]interface{} {
	return map[string]interface{}{
		"region": "us-gov-east-1",
	}
}

func testPodWithoutIRSA() *corev1.Pod {
	return testPodWithEnv(nil)
}

func testPodWithIRSA() *corev1.Pod {
	return testPodWithEnv([]corev1.EnvVar{
		{
			Name:  "AWS_ROLE_ARN",
			Value: "foorole",
		},
		{
			Name:  "AWS_WEB_IDENTITY_TOKEN_FILE",
			Value: "footoken",
		},
		{
			Name:  "AWS_DEFAULT_REGION",
			Value: "default-region",
		},
		{
			Name:  "AWS_REGION",
			Value: "test-region",
		},
	})
}

func testPodWithRegionInAuthConfig() *corev1.Pod {
	return testPodWithEnv([]corev1.EnvVar{
		{
			Name:  "AWS_ROLE_ARN",
			Value: "foorole",
		},
		{
			Name:  "AWS_WEB_IDENTITY_TOKEN_FILE",
			Value: "footoken",
		},
	})
}

func testPodWithEnv(envVars []corev1.EnvVar) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "foobar",
					Env:  envVars,
				},
			},
		},
	}
}

func TestIsBase64(t *testing.T) {

	//Test that the function returns false for a non-base64 encoded string
	notABase64String := `Invalid string`
	got := IsBase64(notABase64String)
	want := false

	if got != want {
		t.Errorf("got %t, wanted %t", got, want)
	}

	//Test that the function returns true for a base64 encoded string
	aBase64String := `VmFsaWQgc3RyaW5nCg==`
	got = IsBase64(aBase64String)
	want = true

	if got != want {
		t.Errorf("got %t, wanted %t", got, want)
	}
}
