package kubernetes

import (
	"bytes"
	"fmt"
	"io"

	"context"

	"github.com/grafana/grafana-app-sdk/resource"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type PodLogger struct {
	ClientSet *kubernetes.Clientset
}

// Get logs for a given pod
// stackoverflow.com/questions/53852530/how-to-get-logs-from-kubernetes-using-go
func (p *PodLogger) Logs(ctx context.Context, ident resource.Identifier) (string, error) {
	podLogOpts := corev1.PodLogOptions{}
	req := p.ClientSet.CoreV1().Pods(ident.Namespace).GetLogs(ident.Name, &podLogOpts)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("error in opening stream")
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", fmt.Errorf("error in copy information from podLogs to buf")
	}
	str := buf.String()

	return str, nil
}

func NewPodLogger(conf *rest.Config) (*PodLogger, error) {
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(conf)
	if err != nil {
		return nil, fmt.Errorf("error in getting access to K8S")
	}

	return &PodLogger{ClientSet: clientset}, nil
}

func (p *PodLogger) LogsForJob(ctx context.Context, j *Job) (string, error) {
	pods, err := p.ClientSet.CoreV1().Pods(j.StaticMeta.Namespace).List(ctx,
		v1.ListOptions{LabelSelector: fmt.Sprintf("job-name=%s", j.StaticMeta.Name)},
	)
	if err != nil {
		return "", err
	}

	// FIXME hardcode for now
	if len(pods.Items) == 0 {
		return "", fmt.Errorf("no pods found for job %s", j.StaticMeta.Name)
	}
	pod := pods.Items[0]

	ident := resource.Identifier{Namespace: pod.ObjectMeta.Namespace, Name: pod.ObjectMeta.Name}

	return p.Logs(ctx, ident)
}

func PodSchema() resource.Schema {
	return resource.NewSimpleSchema("batch", "v1", &Job{}, resource.WithKind("Job"))
}

type Pod struct {
}

//Lookup pod for given job
//kubectl get pods -l job-name=build-grafana-3edeafa-arm64-darwin

//Look up logs for given pod
//kubectl logs build-grafana-3edeafa-arm64-darwin-9fn2p
