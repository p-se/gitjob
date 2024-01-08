package gitjob

import (
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/rancher/gitjob/internal/mocks"
	v1 "github.com/rancher/gitjob/pkg/apis/gitjob.cattle.io/v1"
	corev1controller "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGenerateJob(t *testing.T) {
	ctrl := gomock.NewController(t)

	securityContext := &corev1.SecurityContext{
		AllowPrivilegeEscalation: &[]bool{false}[0],
		ReadOnlyRootFilesystem:   &[]bool{true}[0],
		Privileged:               &[]bool{false}[0],
		Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
		RunAsNonRoot:             &[]bool{true}[0],
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}

	tests := map[string]struct {
		gitjob                 *v1.GitJob
		secret                 corev1controller.SecretCache
		expectedInitContainers []corev1.Container
		expectedVolumes        []corev1.Volume
		expectedErr            error
	}{
		"simple (no credentials, no ca, no skip tls)": {
			gitjob: &v1.GitJob{
				Spec: v1.GitJobSpec{Git: v1.GitInfo{Repo: "repo"}},
			},
			expectedInitContainers: []corev1.Container{
				{
					Command: []string{
						"gitcloner",
					},
					Args:  []string{"repo", "/workspace"},
					Image: "test",
					Name:  "gitcloner-initializer",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      gitClonerVolumeName,
							MountPath: "/workspace",
						},
						{
							Name:      emptyDirVolumeName,
							MountPath: "/tmp",
						},
					},
					SecurityContext: securityContext,
				},
			},
			expectedVolumes: []corev1.Volume{
				{
					Name: gitClonerVolumeName,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: emptyDirVolumeName,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
		"http credentials": {
			gitjob: &v1.GitJob{
				Spec: v1.GitJobSpec{
					Git: v1.GitInfo{
						Repo: "repo",
						Credential: v1.Credential{
							ClientSecretName: "secretName",
						},
					},
				},
			},
			expectedInitContainers: []corev1.Container{
				{
					Command: []string{
						"gitcloner",
					},
					Args:  []string{"repo", "/workspace", "--username", "user", "--password-file", "/gitjob/credentials/" + corev1.BasicAuthPasswordKey},
					Image: "test",
					Name:  "gitcloner-initializer",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      gitClonerVolumeName,
							MountPath: "/workspace",
						},
						{
							Name:      emptyDirVolumeName,
							MountPath: "/tmp",
						},
						{
							Name:      gitCredentialVolumeName,
							MountPath: "/gitjob/credentials",
						},
					},
					SecurityContext: securityContext,
				},
			},
			expectedVolumes: []corev1.Volume{
				{
					Name: gitClonerVolumeName,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: emptyDirVolumeName,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: gitCredentialVolumeName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: "secretName",
						},
					},
				},
			},
			secret: httpSecretMock(ctrl),
		},
		"ssh credentials": {
			gitjob: &v1.GitJob{
				Spec: v1.GitJobSpec{
					Git: v1.GitInfo{
						Repo: "repo",
						Credential: v1.Credential{
							ClientSecretName: "secretName",
						},
					},
				},
			},
			expectedInitContainers: []corev1.Container{
				{
					Command: []string{
						"gitcloner",
					},
					Args:  []string{"repo", "/workspace", "--ssh-private-key-file", "/gitjob/ssh/" + corev1.SSHAuthPrivateKey},
					Image: "test",
					Name:  "gitcloner-initializer",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      gitClonerVolumeName,
							MountPath: "/workspace",
						},
						{
							Name:      emptyDirVolumeName,
							MountPath: "/tmp",
						},
						{
							Name:      gitCredentialVolumeName,
							MountPath: "/gitjob/ssh",
						},
					},
					SecurityContext: securityContext,
				},
			},
			expectedVolumes: []corev1.Volume{
				{
					Name: gitClonerVolumeName,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: emptyDirVolumeName,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: gitCredentialVolumeName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: "secretName",
						},
					},
				},
			},
			secret: sshSecretMock(ctrl),
		},
		"custom CA": {
			gitjob: &v1.GitJob{
				Spec: v1.GitJobSpec{
					Git: v1.GitInfo{
						Credential: v1.Credential{
							CABundle: []byte("ca"),
						},
						Repo: "repo",
					},
				},
			},
			expectedInitContainers: []corev1.Container{
				{
					Command: []string{
						"gitcloner",
					},
					Args:  []string{"repo", "/workspace", "--ca-bundle-file", "/gitjob/cabundle/" + bundleCAFile},
					Image: "test",
					Name:  "gitcloner-initializer",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      gitClonerVolumeName,
							MountPath: "/workspace",
						},
						{
							Name:      emptyDirVolumeName,
							MountPath: "/tmp",
						},
						{
							Name:      bundleCAVolumeName,
							MountPath: "/gitjob/cabundle",
						},
					},
					SecurityContext: securityContext,
				},
			},
			expectedVolumes: []corev1.Volume{
				{
					Name: gitClonerVolumeName,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: emptyDirVolumeName,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: bundleCAVolumeName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: "-cabundle",
						},
					},
				},
			},
		},
		"skip tls": {
			gitjob: &v1.GitJob{
				Spec: v1.GitJobSpec{
					Git: v1.GitInfo{
						Credential: v1.Credential{
							InsecureSkipTLSverify: true,
						},
						Repo: "repo",
					},
				},
			},
			expectedInitContainers: []corev1.Container{
				{
					Command: []string{
						"gitcloner",
					},
					Args:  []string{"repo", "/workspace", "--insecure-skip-tls"},
					Image: "test",
					Name:  "gitcloner-initializer",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      gitClonerVolumeName,
							MountPath: "/workspace",
						},
						{
							Name:      emptyDirVolumeName,
							MountPath: "/tmp",
						},
					},
					SecurityContext: securityContext,
				},
			},
			expectedVolumes: []corev1.Volume{
				{
					Name: gitClonerVolumeName,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: emptyDirVolumeName,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			h := Handler{
				image:   "test",
				secrets: test.secret,
			}
			job, err := h.generateJob(test.gitjob)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !cmp.Equal(job.Spec.Template.Spec.InitContainers, test.expectedInitContainers) {
				t.Fatalf("expected initContainers: %v, got: %v", test.expectedInitContainers, job.Spec.Template.Spec.InitContainers)
			}
			if !cmp.Equal(job.Spec.Template.Spec.Volumes, test.expectedVolumes) {
				t.Fatalf("expected volumes: %v, got: %v", test.expectedVolumes, job.Spec.Template.Spec.Volumes)
			}
		})
	}
}

func TestGenerateJob_EnvVars(t *testing.T) {
	tests := map[string]struct {
		gitjob                       *v1.GitJob
		osEnv                        map[string]string
		expectedContainerEnvVars     []corev1.EnvVar
		expectedInitContainerEnvVars []corev1.EnvVar
	}{
		"no proxy": {
			gitjob: &v1.GitJob{
				Spec: v1.GitJobSpec{
					JobSpec: batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Env: []corev1.EnvVar{{
											Name:  "foo",
											Value: "bar",
										}},
									},
								},
							},
						},
					},
				},
				Status: v1.GitJobStatus{
					GitEvent: v1.GitEvent{
						Commit: "commit",
						GithubMeta: v1.GithubMeta{
							Event: "event",
						},
					},
				},
			},
			expectedContainerEnvVars: []corev1.EnvVar{
				{
					Name:  "foo",
					Value: "bar",
				},
				{
					Name:  "COMMIT",
					Value: "commit",
				},
				{
					Name:  "EVENT_TYPE",
					Value: "event",
				},
			},
		},
		"proxy": {
			gitjob: &v1.GitJob{
				Spec: v1.GitJobSpec{
					JobSpec: batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Env: []corev1.EnvVar{{
											Name:  "foo",
											Value: "bar",
										}},
									},
								},
							},
						},
					},
				},
				Status: v1.GitJobStatus{
					GitEvent: v1.GitEvent{
						Commit: "commit",
						GithubMeta: v1.GithubMeta{
							Event: "event",
						},
					},
				},
			},
			expectedContainerEnvVars: []corev1.EnvVar{
				{
					Name:  "foo",
					Value: "bar",
				},
				{
					Name:  "COMMIT",
					Value: "commit",
				},
				{
					Name:  "EVENT_TYPE",
					Value: "event",
				},
				{
					Name:  "HTTP_PROXY",
					Value: "httpProxy",
				},
				{
					Name:  "HTTPS_PROXY",
					Value: "httpsProxy",
				},
			},
			expectedInitContainerEnvVars: []corev1.EnvVar{
				{
					Name:  "HTTP_PROXY",
					Value: "httpProxy",
				},
				{
					Name:  "HTTPS_PROXY",
					Value: "httpsProxy",
				},
			},
			osEnv: map[string]string{"HTTP_PROXY": "httpProxy", "HTTPS_PROXY": "httpsProxy"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			h := Handler{
				image: "test",
			}
			for k, v := range test.osEnv {
				err := os.Setenv(k, v)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
			job, err := h.generateJob(test.gitjob)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !cmp.Equal(job.Spec.Template.Spec.Containers[0].Env, test.expectedContainerEnvVars) {
				t.Errorf("unexpected envVars. expected %v, but got %v", test.expectedContainerEnvVars, job.Spec.Template.Spec.Containers[0].Env)
			}
			if !cmp.Equal(job.Spec.Template.Spec.InitContainers[0].Env, test.expectedInitContainerEnvVars) {
				t.Errorf("unexpected envVars. expected %v, but got %v", test.expectedInitContainerEnvVars, job.Spec.Template.Spec.InitContainers[0].Env)
			}
			for k := range test.osEnv {
				err := os.Unsetenv(k)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func httpSecretMock(ctrl *gomock.Controller) corev1controller.SecretCache {
	secretmock := mocks.NewMockSecretCache(ctrl)
	secretmock.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{},
		Data: map[string][]byte{
			corev1.BasicAuthUsernameKey: []byte("user"),
			corev1.BasicAuthPasswordKey: []byte("pass"),
		},
		Type: corev1.SecretTypeBasicAuth,
	}, nil)

	return secretmock
}

func sshSecretMock(ctrl *gomock.Controller) corev1controller.SecretCache {
	secretmock := mocks.NewMockSecretCache(ctrl)
	secretmock.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{},
		Data: map[string][]byte{
			corev1.SSHAuthPrivateKey: []byte("ssh key"),
		},
		Type: corev1.SecretTypeSSHAuth,
	}, nil)

	return secretmock
}