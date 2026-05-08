package controller

import (
	"testing"

	corev1 "k8s.io/api/core/v1"

	inferencev1alpha1 "github.com/defilantech/llmkube/api/v1alpha1"
)

func TestOpenVINOBackendDefaults(t *testing.T) {
	backend := &OpenVINOBackend{}

	if got := backend.ContainerName(); got != "openvino" {
		t.Fatalf("ContainerName() = %q, want %q", got, "openvino")
	}
	if got := backend.DefaultImage(); got != "openvino/model_server:latest" {
		t.Fatalf("DefaultImage() = %q, want %q", got, "openvino/model_server:latest")
	}
	if got := backend.DefaultPort(); got != int32(8000) {
		t.Fatalf("DefaultPort() = %d, want %d", got, 8000)
	}
	if got := backend.NeedsModelInit(); !got {
		t.Fatalf("NeedsModelInit() = %v, want true", got)
	}
}

func TestOpenVINOBuildArgs(t *testing.T) {
	backend := &OpenVINOBackend{}

	isvc := &inferencev1alpha1.InferenceService{
		Spec: inferencev1alpha1.InferenceServiceSpec{
			ExtraArgs: []string{"--layout", "NCHW"},
			OpenVINOConfig: &inferencev1alpha1.OpenVINOConfig{
				ModelName:    "phi4",
				TargetDevice: "GPU",
			},
		},
	}
	model := &inferencev1alpha1.Model{Spec: inferencev1alpha1.ModelSpec{Source: "ignored-when-model-path-present"}}

	args := backend.BuildArgs(isvc, model, "/models/phi4", 8000)

	expectedPairs := [][2]string{
		{"--model_name", "phi4"},
		{"--model_path", "/models"},
		{"--rest_port", "8000"},
		{"--port", "9000"},
		{"--target_device", "GPU"},
		{"--layout", "NCHW"},
	}

	for _, pair := range expectedPairs {
		if !containsArg(args, pair[0], pair[1]) {
			t.Fatalf("BuildArgs() missing %s %s in %v", pair[0], pair[1], args)
		}
	}
}

func TestOpenVINOBuildArgsUsesModelSourceFallback(t *testing.T) {
	backend := &OpenVINOBackend{}
	isvc := &inferencev1alpha1.InferenceService{}
	model := &inferencev1alpha1.Model{Spec: inferencev1alpha1.ModelSpec{Source: "https://example.com/model"}}

	args := backend.BuildArgs(isvc, model, "", 8000)
	if !containsArg(args, "--model_path", "https://example.com/model") {
		t.Fatalf("BuildArgs() should fallback to model source, args=%v", args)
	}
}

func TestOpenVINOBuildEnv(t *testing.T) {
	backend := &OpenVINOBackend{}
	isvc := &inferencev1alpha1.InferenceService{
		Spec: inferencev1alpha1.InferenceServiceSpec{
			OpenVINOConfig: &inferencev1alpha1.OpenVINOConfig{
				HFTokenSecretRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "hf-token"},
					Key:                  "HF_TOKEN",
				},
			},
		},
	}

	env := backend.BuildEnv(isvc)
	if len(env) != 1 {
		t.Fatalf("BuildEnv() len = %d, want 1", len(env))
	}
	if env[0].Name != "HF_TOKEN" {
		t.Fatalf("BuildEnv()[0].Name = %q, want %q", env[0].Name, "HF_TOKEN")
	}
	if env[0].ValueFrom == nil || env[0].ValueFrom.SecretKeyRef == nil {
		t.Fatalf("BuildEnv()[0] should contain SecretKeyRef")
	}
	if env[0].ValueFrom.SecretKeyRef.Name != "hf-token" {
		t.Fatalf("BuildEnv() secret name = %q, want %q", env[0].ValueFrom.SecretKeyRef.Name, "hf-token")
	}
}

func TestResolveBackendOpenVINO(t *testing.T) {
	isvc := &inferencev1alpha1.InferenceService{Spec: inferencev1alpha1.InferenceServiceSpec{Runtime: RuntimeOpenVINO}}
	backend := resolveBackend(isvc)
	if _, ok := backend.(*OpenVINOBackend); !ok {
		t.Fatalf("resolveBackend(openvino) did not return OpenVINOBackend")
	}
}
