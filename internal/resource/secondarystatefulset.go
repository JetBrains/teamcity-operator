package resource

//
//import (
//	"context"
//	"git.jetbrains.team/tch/teamcity-operator/internal/metadata"
//	"sigs.k8s.io/controller-runtime/pkg/client"
//)
//
//type SecondaryStatefulSetBuilder struct {
//	*TeamCityResourceBuilder
//}
//
//func (builder SecondaryStatefulSetBuilder) BuildObjectList() ([]client.Object, error) {
//	var objectList []client.Object
//	for _, secondaryNode := range builder.Instance.Spec.SecondaryNodes {
//		nodeLabels := metadata.GetLabels(secondaryNode.Name, builder.Instance.Labels)
//		node := CreateEmptyStatefulSet(secondaryNode.Name, builder.Instance.Namespace, nodeLabels)
//		objectList = append(objectList, &node)
//	}
//	return objectList, nil
//}
//
//func (builder SecondaryStatefulSetBuilder) Update(object client.Object) error {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (builder SecondaryStatefulSetBuilder) GetObsoleteObjects(ctx context.Context) ([]client.Object, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (builder SecondaryStatefulSetBuilder) UpdateMayRequireStsRecreate() bool {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (builder *TeamCityResourceBuilder) SecondaryStatefulSet() *SecondaryStatefulSetBuilder {
//	return &SecondaryStatefulSetBuilder{builder}
//}
