package predicate

import (
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func TeamcityEventPredicates() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(createEvent event.CreateEvent) bool {
			return true
		},
		UpdateFunc: func(updateEvent event.UpdateEvent) bool {
			return true
		},
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			return !deleteEvent.DeleteStateUnknown
		},
		GenericFunc: func(genericEvent event.GenericEvent) bool {
			return true
		},
	}
}

func StatefulSetEventPredicates() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(createEvent event.CreateEvent) bool {
			return true
		},
		UpdateFunc: func(updateEvent event.UpdateEvent) bool {
			return shouldFilterOutUpdateEventForStatefulSet(updateEvent)
		},
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			return !deleteEvent.DeleteStateUnknown
		},
		GenericFunc: func(genericEvent event.GenericEvent) bool {
			return true
		},
	}
}

func shouldFilterOutUpdateEventForStatefulSet(event event.UpdateEvent) bool {
	//attempt to cast updated object to StatefulSet
	//if casting fails - controller should skip this event
	oldStatefulSet, ok := event.ObjectOld.(*v1.StatefulSet)
	if !ok {
		return false
	}
	newStatefulSet, ok := event.ObjectNew.(*v1.StatefulSet)
	if !ok {
		return false
	}
	//if spec of StatefulSet did not change, the event is ignored
	if equal(oldStatefulSet.Spec, newStatefulSet.Spec) {
		return false
	}
	return true
}

func equal(x, y interface{}) bool {
	//DeepEqual is not always capable of detecting identical objects due to various defaults and type conversions
	//DeepDerivative is more reliable way since it does not care for defaults and types as much
	return reflect.DeepEqual(x, y) || equality.Semantic.DeepDerivative(x, y)
}
