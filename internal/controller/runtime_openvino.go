package controller

import (
	"fmt"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	inferencev1alpha1 "github.com/defilantech/llmkube/api/v1alpha1"
)

const (
	// RuntimeOpenVINO selects the Intel OpenVINO backend.
	RuntimeOpenVINO = "openvino"
)

// OpenVINOBackend generates container configuration for Intel OpenVINO Model Server.
type OpenVINOBackend struct{}

func (b *OpenVINOBackend) ContainerName() string { return "openvino" }

func (b *OpenVINOBackend) DefaultImage() string {
	return "openvino/model_server:latest"
}

func (b *OpenVINOBackend) DefaultPort() int32       { return 8000 }
func (b *OpenVINOBackend) NeedsModelInit() bool     { return true }
func (b *OpenVINOBackend) DefaultHPAMetric() string { return "" }

func (b *OpenVINOBackend) BuildArgs(isvc *inferencev1alpha1.InferenceService, model *inferencev1alpha1.Model, modelPath string, port int32) []string {
	modelDir := modelPath
	if modelDir != "" {
		modelDir = filepath.Dir(modelDir)
	} else {
		modelDir = model.Spec.Source
	}

	modelName := model.Name
	if isvc.Spec.OpenVINOConfig != nil && isvc.Spec.OpenVINOConfig.ModelName != "" {
		modelName = isvc.Spec.OpenVINOConfig.ModelName
	}

	args := []string{
		"--model_name", modelName,
		"--model_path", modelDir,
		"--rest_port", fmt.Sprintf("%d", port),
		"--port", "9000",
	}

	if isvc.Spec.OpenVINOConfig != nil && isvc.Spec.OpenVINOConfig.TargetDevice != "" {
		args = append(args, "--target_device", isvc.Spec.OpenVINOConfig.TargetDevice)
	}

	if len(isvc.Spec.ExtraArgs) > 0 {
		args = append(args, isvc.Spec.ExtraArgs...)
	}

	return args
}

func (b *OpenVINOBackend) BuildProbes(port int32) (startup, liveness, readiness *corev1.Probe) {
	startup = &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromInt32(port)},
		},
		PeriodSeconds:    10,
		TimeoutSeconds:   5,
		FailureThreshold: 180,
	}

	liveness = &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromInt32(port)},
		},
		PeriodSeconds:    15,
		TimeoutSeconds:   5,
		FailureThreshold: 3,
	}

	readiness = &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromInt32(port)},
		},
		PeriodSeconds:    10,
		TimeoutSeconds:   5,
		FailureThreshold: 3,
	}

	return startup, liveness, readiness
}

func (b *OpenVINOBackend) BuildEnv(isvc *inferencev1alpha1.InferenceService) []corev1.EnvVar {
	if isvc.Spec.OpenVINOConfig != nil && isvc.Spec.OpenVINOConfig.HFTokenSecretRef != nil {
		return []corev1.EnvVar{{
			Name:      "HF_TOKEN",
			ValueFrom: &corev1.EnvVarSource{SecretKeyRef: isvc.Spec.OpenVINOConfig.HFTokenSecretRef},
		}}
	}

	return nil
}