/*
 Copyright 2021 The Hybridnet Authors.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package controller

import (
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	multiclusterv1 "github.com/alibaba/hybridnet/pkg/apis/multicluster/v1"
	"github.com/alibaba/hybridnet/pkg/constants"
)

// add handler for RemoteVtep and RemoteSubnet

type enqueueRequestForRemoteVtep struct {
	handler.Funcs
}

func (e enqueueRequestForRemoteVtep) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	q.Add(ActionReconcileNode)
}

func (e enqueueRequestForRemoteVtep) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	oldRemoteVtep := evt.ObjectOld.(*multiclusterv1.RemoteVtep)
	newRemoteVtep := evt.ObjectNew.(*multiclusterv1.RemoteVtep)

	if oldRemoteVtep.Spec.VTEPInfo.IP != newRemoteVtep.Spec.VTEPInfo.IP ||
		oldRemoteVtep.Spec.VTEPInfo.MAC != newRemoteVtep.Spec.VTEPInfo.MAC ||
		oldRemoteVtep.Annotations[constants.AnnotationNodeLocalVxlanIPList] != newRemoteVtep.Annotations[constants.AnnotationNodeLocalVxlanIPList] ||
		!isIPListEqual(oldRemoteVtep.Spec.EndpointIPList, newRemoteVtep.Spec.EndpointIPList) {
		q.Add(ActionReconcileNode)
	}
}

func (e enqueueRequestForRemoteVtep) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	q.Add(ActionReconcileNode)
}
