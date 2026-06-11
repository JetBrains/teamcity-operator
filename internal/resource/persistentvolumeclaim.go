package resource

import (
	"context"
	"errors"
	"fmt"
	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	"git.jetbrains.team/tch/teamcity-operator/internal/metadata"
	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type PersistentVolumeClaimBuilder struct {
	*TeamCityResourceBuilder
}

func (builder *TeamCityResourceBuilder) PersistentVolumeClaim() *PersistentVolumeClaimBuilder {
	return &PersistentVolumeClaimBuilder{builder}
}

func (builder PersistentVolumeClaimBuilder) BuildObjectList() ([]client.Object, error) {
	var objectList []client.Object
	objectList = append(objectList, &v12.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: builder.Instance.Spec.DataDirVolumeClaim.Name, Namespace: builder.Instance.Namespace},
	})
	for _, pvc := range builder.Instance.Spec.PersistentVolumeClaims {
		objectList = append(objectList, &v12.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: pvc.Name, Namespace: builder.Instance.Namespace},
		})
	}
	return objectList, nil
}

func (builder PersistentVolumeClaimBuilder) Update(object client.Object) error {
	var idx int
	var pvcList []CustomPersistentVolumeClaim
	pvcList = append(pvcList, builder.Instance.Spec.DataDirVolumeClaim)
	pvcList = append(pvcList, builder.Instance.Spec.PersistentVolumeClaims...)
	if idx = builder.getPVCIndex(object, pvcList); idx == -1 {
		return fmt.Errorf("failed to update object: %w", errors.New("the specified PVC does not exist: "+object.GetName()))
	}

	desired := pvcList[idx]
	persistentVolumeClaim := object.(*v12.PersistentVolumeClaim)
	persistentVolumeClaim.Annotations = desired.Annotations
	persistentVolumeClaim.Labels = metadata.GetLabels(builder.Instance.Name, builder.Instance.Labels)
	persistentVolumeClaim.Spec.AccessModes = desired.Spec.AccessModes
	persistentVolumeClaim.Spec.Selector = desired.Spec.Selector
	persistentVolumeClaim.Spec.Resources = desired.Spec.Resources
	if len(persistentVolumeClaim.Spec.VolumeName) < 0 {
		persistentVolumeClaim.Spec.VolumeName = desired.Spec.VolumeName
	}
	if persistentVolumeClaim.Spec.StorageClassName == nil {
		persistentVolumeClaim.Spec.StorageClassName = desired.Spec.StorageClassName
	}
	if persistentVolumeClaim.Spec.VolumeMode == nil {
		persistentVolumeClaim.Spec.VolumeMode = desired.Spec.VolumeMode
	}
	if persistentVolumeClaim.Spec.DataSource == nil {
		persistentVolumeClaim.Spec.DataSource = desired.Spec.DataSource
	}
	if persistentVolumeClaim.Spec.DataSourceRef == nil {
		persistentVolumeClaim.Spec.DataSourceRef = desired.Spec.DataSourceRef
	}
	if err := controllerutil.SetControllerReference(builder.Instance, persistentVolumeClaim, builder.Scheme); err != nil {
		return fmt.Errorf("failed setting controller reference: %w", err)
	}
	return nil
}

func (builder PersistentVolumeClaimBuilder) GetObsoleteObjects(ctx context.Context) ([]client.Object, error) {
	currentPVCList := &v12.PersistentVolumeClaimList{}
	var obsoleteObjects []client.Object
	var pvcList []CustomPersistentVolumeClaim

	listOptions := []client.ListOption{
		client.InNamespace(builder.Instance.Namespace),
		client.MatchingLabels(metadata.GetLabels(builder.Instance.Name, builder.Instance.Labels)),
	}
	if err := builder.Client.List(ctx, currentPVCList, listOptions...); err != nil {
		return nil, err
	}

	pvcList = append(pvcList, builder.Instance.Spec.DataDirVolumeClaim)
	pvcList = append(pvcList, builder.Instance.Spec.PersistentVolumeClaims...)
	for _, pvc := range currentPVCList.Items {
		var idx int
		s := pvc
		if idx = builder.getPVCIndex(&pvc, pvcList); idx == -1 {
			obsoleteObjects = append(obsoleteObjects, &s)
		}
	}
	return obsoleteObjects, nil
}

func (builder PersistentVolumeClaimBuilder) UpdateMayRequireStsRecreate() bool {
	return false
}

func (builder PersistentVolumeClaimBuilder) getPVCIndex(object client.Object, pvcList []CustomPersistentVolumeClaim) int {
	for idx, pvc := range pvcList {
		if pvc.Name == object.GetName() {
			return idx
		}
	}
	return -1
}
