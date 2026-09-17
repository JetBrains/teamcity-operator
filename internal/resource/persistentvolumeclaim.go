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
	for _, nodePVC := range builder.nodeClaimsToManage() {
		objectList = append(objectList, &v12.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: nodePVC.Name, Namespace: builder.Instance.Namespace},
		})
	}
	return objectList, nil
}

func (builder PersistentVolumeClaimBuilder) Update(object client.Object) error {
	pvcList := builder.allManagedClaims()
	idx := builder.getPVCIndex(object, pvcList)
	if idx == -1 {
		return fmt.Errorf("failed to update object: %w", errors.New("the specified PVC does not exist: "+object.GetName()))
	}

	desired := pvcList[idx]
	persistentVolumeClaim := object.(*v12.PersistentVolumeClaim)
	persistentVolumeClaim.Annotations = desired.Annotations
	labels := metadata.GetLabels(builder.Instance.Name, builder.Instance.Labels)
	if builder.isManagedNodeClaim(object.GetName()) {
		labels = metadata.WithNodePVCLabel(labels)
	}
	persistentVolumeClaim.Labels = labels
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
	if err := controllerutil.SetControllerReference(builder.Instance, persistentVolumeClaim, builder.Scheme); err != nil {
		return fmt.Errorf("failed setting controller reference: %w", err)
	}
	return nil
}

func (builder PersistentVolumeClaimBuilder) GetObsoleteObjects(ctx context.Context) ([]client.Object, error) {
	currentPVCList := &v12.PersistentVolumeClaimList{}
	var obsoleteObjects []client.Object

	listOptions := []client.ListOption{
		client.InNamespace(builder.Instance.Namespace),
		client.MatchingLabels(metadata.GetLabels(builder.Instance.Name, builder.Instance.Labels)),
	}
	if err := builder.Client.List(ctx, currentPVCList, listOptions...); err != nil {
		return nil, err
	}

	desiredNames := builder.desiredClaimNamesForObsoleteCheck()
	retainedNodeClaimNames := builder.retainedNodeClaimNames()
	for _, pvc := range currentPVCList.Items {
		s := pvc
		if metadata.IsNodePVC(pvc.Labels) {
			continue
		}
		if _, retained := retainedNodeClaimNames[pvc.Name]; retained {
			continue
		}
		if _, desired := desiredNames[pvc.Name]; !desired {
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

func (builder PersistentVolumeClaimBuilder) allManagedClaims() []CustomPersistentVolumeClaim {
	pvcList := []CustomPersistentVolumeClaim{builder.Instance.Spec.DataDirVolumeClaim}
	pvcList = append(pvcList, builder.Instance.Spec.PersistentVolumeClaims...)
	pvcList = append(pvcList, builder.nodeClaimsToManage()...)
	return pvcList
}

func (builder PersistentVolumeClaimBuilder) desiredClaimNamesForObsoleteCheck() map[string]struct{} {
	names := map[string]struct{}{
		builder.Instance.Spec.DataDirVolumeClaim.Name: {},
	}
	for _, pvc := range builder.Instance.Spec.PersistentVolumeClaims {
		names[pvc.Name] = struct{}{}
	}
	return names
}

func (builder PersistentVolumeClaimBuilder) retainedNodeClaimNames() map[string]struct{} {
	names := map[string]struct{}{}
	for _, claim := range builder.Instance.AllNodePersistentVolumeClaims() {
		names[claim.Name] = struct{}{}
	}
	return names
}

func (builder PersistentVolumeClaimBuilder) isManagedNodeClaim(name string) bool {
	for _, claim := range builder.nodeClaimsToManage() {
		if claim.Name == name {
			return true
		}
	}
	return false
}

func (builder PersistentVolumeClaimBuilder) nodeClaimsToManage() []CustomPersistentVolumeClaim {
	var claims []CustomPersistentVolumeClaim
	for _, claim := range builder.Instance.AllNodePersistentVolumeClaims() {
		if claim.ExistingClaim {
			continue
		}
		claims = append(claims, claim)
	}
	return claims
}
