package base

import (
	corev1 "k8s.io/api/core/v1"
)

func compareContainerResources(expected corev1.ResourceRequirements, actual corev1.ResourceRequirements) bool {
	expectedRequestCPU, expectedRequestCPUSet := expected.Requests[corev1.ResourceCPU]
	expectedRequestMemory, expectedRequestMemorySet := expected.Requests[corev1.ResourceMemory]
	expectedLimitCPU, expectedLimitCPUSet := expected.Limits[corev1.ResourceCPU]
	expectedLimitMemory, expectedLimitMemorySet := expected.Limits[corev1.ResourceMemory]
	actualRequestCPU, actualRequestCPUSet := actual.Requests[corev1.ResourceCPU]
	actualRequestMemory, actualRequestMemorySet := actual.Requests[corev1.ResourceMemory]
	actualLimitCPU, actualLimitCPUSet := actual.Limits[corev1.ResourceCPU]
	actualLimitMemory, actualLimitMemorySet := actual.Limits[corev1.ResourceMemory]

	if expectedRequestCPUSet && (!actualRequestCPUSet || expectedRequestCPU.String() != actualRequestCPU.String()) {
		return false
	}
	if expectedRequestMemorySet && (!actualRequestMemorySet || expectedRequestMemory.String() != actualRequestMemory.String()) {
		return false
	}
	if expectedLimitCPUSet && (!actualLimitCPUSet || expectedLimitCPU.String() != actualLimitCPU.String()) {
		return false
	}
	if expectedLimitMemorySet && (!actualLimitMemorySet || expectedLimitMemory.String() != actualLimitMemory.String()) {
		return false
	}
	return true
}
