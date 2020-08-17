package kubernetes

import (
	"sync"
	"time"

	k8sV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sInformers "k8s.io/client-go/informers"
	k8sClient "k8s.io/client-go/kubernetes"
	k8sCache "k8s.io/client-go/tools/cache"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
)

const informerCoolDown = time.Millisecond * 250

// concurrentSlice is used to track internal state of the informer in a thread safe way.
type concurrentSlice struct {
	lock    sync.Mutex
	updates []*k8sV1.Pod
}

func newConcurrentSlice() *concurrentSlice {
	return &concurrentSlice{updates: make([]*k8sV1.Pod, 0)}
}

func (c *concurrentSlice) append(pod *k8sV1.Pod) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.updates = append(c.updates, pod)
}

func (c *concurrentSlice) getUpdates() []*k8sV1.Pod {
	c.lock.Lock()
	defer c.lock.Unlock()
	if len(c.updates) == 0 {
		return nil
	}

	updates := c.updates
	c.updates = make([]*k8sV1.Pod, 0)
	return updates
}

// messages that are sent to the informer.
type (
	informerTick struct{}
)

// messages that are sent by the informer.
type (
	podStatusUpdate struct {
		updatedPod *k8sV1.Pod
	}
)

type informer struct {
	clientSet   *k8sClient.Clientset
	namespace   string
	podsHandler *actor.Ref

	updates    *concurrentSlice
	stopSignal chan struct{}
}

func newInformer(
	clientSet *k8sClient.Clientset,
	namespace string,
	podsHandler *actor.Ref,
) *informer {
	return &informer{
		clientSet:   clientSet,
		namespace:   namespace,
		podsHandler: podsHandler,
		updates:     newConcurrentSlice(),
		stopSignal:  make(chan struct{}),
	}
}

// Receive implements the actor interface.
func (i *informer) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		ctx.Tell(ctx.Self(), informerTick{})

	case informerTick:

	case actor.PostStop:
		// This should never be reached.
		close(i.stopSignal)

	default:
		ctx.Log().Errorf("unexpected message %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (i *informer) prepareInformer(ctx *actor.Context) {
	k8sInformers.NewSharedInformerFactory(i.clientSet, 0)

	// Set up options to filter pods.
	options := func(options *metaV1.ListOptions) {
		options.LabelSelector = determinedLabel
	}
	sharedOptions := []k8sInformers.SharedInformerOption{
		k8sInformers.WithNamespace(i.namespace),
		k8sInformers.WithTweakListOptions(options),
	}

	informer := k8sInformers.NewSharedInformerFactoryWithOptions(
		i.clientSet, time.Second*5, sharedOptions...)
	podInformer := informer.Core().V1().Pods().Informer()
	podInformer.AddEventHandler(&k8sCache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			newPod := obj.(*k8sV1.Pod)
			i.updates.append(newPod)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			updatedPod := newObj.(*k8sV1.Pod)
			i.updates.append(updatedPod)
		},
		DeleteFunc: func(obj interface{}) {
			deletedPod := obj.(*k8sV1.Pod)
			i.updates.append(deletedPod)
		},
	})

	ctx.Log().Infof("starting pod informer")
	informer.Start(i.stopSignal)
	for !podInformer.HasSynced() {
	}
	ctx.Log().Infof("pod informer has synced")
}

func (i *informer) processUpdates(ctx *actor.Context) {
	for _, update := range i.updates.getUpdates() {
		ctx.Tell(i.podsHandler, podStatusUpdate{updatedPod: update})
	}
	actors.NotifyAfter(ctx, informerCoolDown, informerTick{})
}
