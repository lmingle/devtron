package kubernetesResourceAuditLogs

import (
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	client "github.com/devtron-labs/devtron/api/helm-app"
	application2 "github.com/devtron-labs/devtron/client/k8s/application"
	"github.com/devtron-labs/devtron/internal/sql/repository/app"
	repository2 "github.com/devtron-labs/devtron/pkg/cluster/repository"
	"github.com/devtron-labs/devtron/pkg/kubernetesResourceAuditLogs/repository"
	"github.com/devtron-labs/devtron/pkg/sql"
	"go.uber.org/zap"
	"time"
)

const (
	delete string = "delete"
	helm   string = "helm"
	GitOps string = "argo_cd"
)

type K8sResourceHistoryService interface {
	SaveArgoCdAppsResourceDeleteHistory(query *application.ApplicationResourceDeleteRequest, appId int, envId int, userId int32) error
	SaveHelmAppsResourceHistory(appIdentifier *client.AppIdentifier, k8sRequestBean *application2.K8sRequestBean, userId int32, actionType string) error
}

type K8sResourceHistoryServiceImpl struct {
	appRepository                app.AppRepository
	K8sResourceHistoryRepository repository.K8sResourceHistoryRepository
	logger                       *zap.SugaredLogger
	envRepository                repository2.EnvironmentRepository
}

func Newk8sResourceHistoryServiceImpl(K8sResourceHistoryRepository repository.K8sResourceHistoryRepository,
	logger *zap.SugaredLogger, appRepository app.AppRepository, envRepository repository2.EnvironmentRepository) *K8sResourceHistoryServiceImpl {
	return &K8sResourceHistoryServiceImpl{
		K8sResourceHistoryRepository: K8sResourceHistoryRepository,
		logger:                       logger,
		appRepository:                appRepository,
		envRepository:                envRepository,
	}
}

func (impl K8sResourceHistoryServiceImpl) SaveArgoCdAppsResourceDeleteHistory(query *application.ApplicationResourceDeleteRequest, appId int, envId int, userId int32) error {

	k8sResourceHistory := repository.K8sResourceHistory{
		AppId:        appId,
		AppName:      *query.Name,
		EnvId:        envId,
		Namespace:    *query.Namespace,
		ResourceName: *query.ResourceName,
		Kind:         *query.Kind,
		Group:        *query.Group,
		ForceDelete:  *query.Force,
		AuditLog: sql.AuditLog{
			UpdatedBy: userId,
			UpdatedOn: time.Now(),
		},
		ActionType:        delete,
		DeploymentAppType: GitOps,
	}

	err := impl.K8sResourceHistoryRepository.SaveK8sResourceHistory(&k8sResourceHistory)

	if err != nil {
		return err
	}

	return nil

}

func (impl K8sResourceHistoryServiceImpl) SaveHelmAppsResourceHistory(appIdentifier *client.AppIdentifier, k8sRequestBean *application2.K8sRequestBean, userId int32, actionType string) error {

	app, err := impl.appRepository.FindActiveByName(appIdentifier.ReleaseName)

	env, err := impl.envRepository.FindOneByNamespaceAndClusterId(appIdentifier.Namespace, appIdentifier.ClusterId)

	k8sResourceHistory := repository.K8sResourceHistory{
		AppId:        app.Id,
		AppName:      appIdentifier.ReleaseName,
		EnvId:        env.Id,
		Namespace:    appIdentifier.Namespace,
		ResourceName: k8sRequestBean.ResourceIdentifier.Name,
		Kind:         k8sRequestBean.ResourceIdentifier.GroupVersionKind.Kind,
		Group:        k8sRequestBean.ResourceIdentifier.GroupVersionKind.Group,
		ForceDelete:  false,
		AuditLog: sql.AuditLog{
			UpdatedBy: userId,
			UpdatedOn: time.Now(),
		},
		ActionType:        actionType,
		DeploymentAppType: helm,
	}

	err = impl.K8sResourceHistoryRepository.SaveK8sResourceHistory(&k8sResourceHistory)

	return err

}
