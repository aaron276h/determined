package kubernetes

import (
	"github.com/pkg/errors"

	k8sV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/determined-ai/determined/master/pkg/actor"
)

// messages that are sent to the informer.
type (
	startInformer struct{}
)

// messages that are sent by the informer.
type (
	podStatusUpdate struct {
		podName    string
		updatedPod *k8sV1.Pod
	}
)

type informer struct {
	podInterface typedV1.PodInterface
	namespace    string
	podsHandler  *actor.Ref
}

func newInformer(
	podInterface typedV1.PodInterface,
	namespace string,
	podsHandler *actor.Ref,
) *informer {
	return &informer{
		podInterface: podInterface,
		namespace:    namespace,
		podsHandler:  podsHandler,
	}
}

func (i *informer) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:

	case startInformer:
		if err := i.receiveStartInformer(ctx); err != nil {
			return err
		}

	case actor.PostStop:

	default:
		ctx.Log().Errorf("unexpected message %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (i *informer) receiveStartInformer(ctx *actor.Context) error {
	watch, err := i.podInterface.Watch(metaV1.ListOptions{LabelSelector: determinedLabel})
	if err != nil {
		return errors.Wrap(err, "error initializing pod watch")
	}

	ctx.Log().Info("pod informer is starting")
	for event := range watch.ResultChan() {
		pod := event.Object.(*k8sV1.Pod)
		ctx.Log().Debugf("informer got new pod event for pod: %s %s", pod.Name, pod.Status.Phase)
		ctx.Tell(i.podsHandler, podStatusUpdate{podName: pod.Name, updatedPod: pod})
	}

	return errors.Errorf("pod informer stopped unexpectedly")
}
